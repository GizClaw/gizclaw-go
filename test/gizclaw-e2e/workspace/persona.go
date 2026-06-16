package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/GizClaw/gizclaw-go/pkg/audio/codec/mp3"
	"github.com/GizClaw/gizclaw-go/pkg/audio/codec/ogg"
	"github.com/GizClaw/gizclaw-go/pkg/audio/codec/opus"
	"github.com/GizClaw/gizclaw-go/pkg/audio/resampler"
	"github.com/GizClaw/gizclaw-go/pkg/genx"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/gizcli"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

type personaDriver struct {
	cfg                 config
	client              openai.Client
	transport           *chatTransport
	newTransport        func() (*chatTransport, error)
	history             []roundHistory
	reloadAgent         func(context.Context) error
	generateUtterance   func(context.Context, int) (string, error)
	synthesizeAudio     func(context.Context, string) ([]byte, [][]byte, error)
	transcribeAudioFile func(context.Context, string) (string, error)
}

type roundHistory struct {
	UserText      string
	InputASR      string
	Transcript    string
	AssistantText string
}

type roundStats struct {
	Index                    int
	UserText                 string
	InputASR                 string
	Transcript               string
	AssistantText            string
	InputOpusPackets         int
	InputOpusBytes           int
	DownlinkPackets          int
	DownlinkBytes            int
	EventCount               int
	UplinkSend               time.Duration
	FirstTranscriptChunk     time.Duration
	FirstTranscriptBeforeEOS bool
	FirstAssistantTextChunk  time.Duration
	FirstAudioChunk          time.Duration
	ResponseTotal            time.Duration
	WorkspaceTotal           time.Duration
}

func (d *personaDriver) run(ctx context.Context) ([]roundStats, error) {
	var stats []roundStats
	for i := 1; i <= d.cfg.Rounds; i++ {
		stat, err := d.runRound(ctx, i)
		stats = append(stats, stat)
		if err != nil {
			return stats, err
		}
		d.history = append(d.history, roundHistory{
			UserText:      stat.UserText,
			InputASR:      stat.InputASR,
			Transcript:    stat.Transcript,
			AssistantText: stat.AssistantText,
		})
	}
	return stats, nil
}

func (d *personaDriver) close() {
	if d == nil || d.transport == nil {
		return
	}
	d.transport.close()
	d.transport = nil
}

func (d *personaDriver) resetTransport() error {
	if d == nil {
		return fmt.Errorf("persona driver is nil")
	}
	if d.newTransport == nil {
		if d.transport != nil {
			return nil
		}
		return fmt.Errorf("transport factory is required")
	}
	if d.transport != nil {
		d.transport.close()
		d.transport = nil
	}
	transport, err := d.newTransport()
	if err != nil {
		return err
	}
	if transport == nil {
		return fmt.Errorf("transport factory returned nil")
	}
	d.transport = transport
	return nil
}

func (d *personaDriver) runRound(ctx context.Context, index int) (roundStats, error) {
	stat := roundStats{Index: index}

	userText, err := d.nextUtterance(ctx, index)
	if err != nil {
		return stat, err
	}
	stat.UserText = userText

	audio, packets, err := d.synthesizeOpus(ctx, userText)
	if err != nil {
		return stat, err
	}
	stat.InputOpusPackets = len(packets)
	for _, packet := range packets {
		stat.InputOpusBytes += len(packet)
	}
	if len(packets) == 0 {
		return stat, fmt.Errorf("round %d: speech returned no opus audio packets", index)
	}

	inputPath, err := d.writeRoundAudio(index, audio)
	if err != nil {
		return stat, err
	}
	inputASR, err := d.transcribe(ctx, inputPath)
	if err != nil {
		return stat, err
	}
	stat.InputASR = inputASR
	if err := assertTextSimilar("input asr", userText, inputASR, 0.45); err != nil {
		return stat, fmt.Errorf("round %d: %w", index, err)
	}

	if d.reloadAgent != nil {
		if err := d.reloadAgent(ctx); err != nil {
			if !isAgentAlreadyRunning(err) {
				return stat, fmt.Errorf("round %d: reload workspace: %w", index, err)
			}
		}
	}
	if err := d.resetTransport(); err != nil {
		return stat, fmt.Errorf("round %d: open transport: %w", index, err)
	}
	d.transport.drain()
	// Direct stamped Opus packets currently carry timestamps but not stream IDs;
	// the server restores them on the fixed "audio" stream.
	streamID := "audio"
	uplinkStart := time.Now()
	if err := d.transport.sendAudioTurn(ctx, streamID, packets); err != nil {
		return stat, fmt.Errorf("round %d: send audio turn: %w", index, err)
	}
	stat.UplinkSend = time.Since(uplinkStart)
	responseStart := time.Now()

	var transcriptText string
	var assistant strings.Builder
	var downlinkFrames [][]byte
	assistantAudioDone := false
	var settle <-chan time.Time
waitResponse:
	for transcriptText == "" || stat.DownlinkPackets == 0 || !assistantAudioDone || settle != nil {
		select {
		case <-ctx.Done():
			return stat, fmt.Errorf("round %d: wait response: %w", index, ctx.Err())
		case <-settle:
			settle = nil
			if transcriptText != "" && stat.DownlinkPackets > 0 {
				break waitResponse
			}
		case err := <-d.transport.errs:
			return stat, fmt.Errorf("round %d: transport: %w", index, err)
		case received := <-d.transport.events:
			event := received.event
			if event.StreamId != nil && *event.StreamId != streamID {
				continue
			}
			stat.EventCount++
			label := eventLabel(event)
			if event.Type == apitypes.PeerStreamEventTypeEos && label == "assistant" {
				assistantAudioDone = true
				settle = time.After(700 * time.Millisecond)
				continue
			}
			if event.Text == nil || strings.TrimSpace(*event.Text) == "" {
				continue
			}
			textLatency := afterStartLatency(received.receivedAt, responseStart)
			switch label {
			case "transcript":
				if stat.FirstTranscriptChunk == 0 {
					stat.FirstTranscriptChunk, stat.FirstTranscriptBeforeEOS = afterStartLatencyStatus(received.receivedAt, responseStart)
				}
				transcriptText = mergeTranscriptText(transcriptText, *event.Text)
			case "assistant":
				if stat.FirstAssistantTextChunk == 0 {
					stat.FirstAssistantTextChunk = textLatency
				}
				assistant.WriteString(*event.Text)
			}
		case packet := <-d.transport.opusPackets:
			downlinkFrames = append(downlinkFrames, append([]byte(nil), packet.frame...))
			if stat.FirstAudioChunk == 0 {
				stat.FirstAudioChunk = afterStartLatency(packet.receivedAt, responseStart)
			}
			stat.DownlinkPackets++
			stat.DownlinkBytes += len(packet.frame)
		}
	}

	stat.Transcript = strings.TrimSpace(transcriptText)
	stat.AssistantText = strings.TrimSpace(assistant.String())
	stat.ResponseTotal = time.Since(responseStart)
	stat.WorkspaceTotal = time.Since(uplinkStart)
	if stat.AssistantText == "" && len(downlinkFrames) > 0 {
		assistantAudio, err := oggOpusFromPackets(16000, 1, downlinkFrames)
		if err != nil {
			return stat, fmt.Errorf("round %d: encode assistant audio: %w", index, err)
		}
		assistantPath, err := d.writeRoundAudioFile(index, "assistant", assistantAudio)
		if err != nil {
			return stat, err
		}
		assistantASR, err := d.transcribe(ctx, assistantPath)
		if err != nil {
			return stat, fmt.Errorf("round %d: assistant audio asr: %w", index, err)
		}
		stat.AssistantText = assistantASR
	}
	if stat.Transcript == "" {
		return stat, fmt.Errorf("round %d: missing transcript text", index)
	}
	if err := assertTextSimilar("transcript", userText, stat.Transcript, 0.45); err != nil {
		return stat, fmt.Errorf("round %d: %w", index, err)
	}
	if stat.AssistantText == "" {
		return stat, fmt.Errorf("round %d: missing assistant text", index)
	}
	if stat.DownlinkPackets == 0 {
		return stat, fmt.Errorf("round %d: missing downlink audio", index)
	}
	return stat, nil
}

func (d *personaDriver) nextUtterance(ctx context.Context, index int) (string, error) {
	if d.generateUtterance != nil {
		return d.generateUtterance(ctx, index)
	}
	var prompt strings.Builder
	prompt.WriteString(d.cfg.Persona)
	prompt.WriteString("\nGenerate the next user utterance for a voice-link E2E test.")
	prompt.WriteString("\nRules: Simplified Chinese only, 8 to 24 Chinese characters, no markdown, no quotes.")
	if len(d.history) > 0 {
		prompt.WriteString("\nPrior rounds:\n")
		for i, h := range d.history {
			fmt.Fprintf(&prompt, "%d. user=%s transcript=%s assistant=%s\n", i+1, h.UserText, h.Transcript, h.AssistantText)
		}
	}
	fmt.Fprintf(&prompt, "\nRound: %d", index)

	completion, err := d.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(d.cfg.Models.LLM),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You generate deterministic short voice test utterances."),
			openai.UserMessage(prompt.String()),
		},
	})
	if err != nil {
		return "", fmt.Errorf("generate persona utterance: %w", err)
	}
	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("generate persona utterance: no choices")
	}
	text := cleanUtterance(completion.Choices[0].Message.Content)
	if text == "" {
		return "", fmt.Errorf("generate persona utterance: empty text")
	}
	return text, nil
}

func (d *personaDriver) synthesizeOpus(ctx context.Context, text string) ([]byte, [][]byte, error) {
	if d.synthesizeAudio != nil {
		return d.synthesizeAudio(ctx, text)
	}
	speech, err := d.client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Input:          text,
		Model:          openai.SpeechModel(d.cfg.Models.TTS),
		Voice:          openai.AudioSpeechNewParamsVoice(d.cfg.Voice),
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatOpus,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("speech: %w", err)
	}
	defer speech.Body.Close()
	audio, err := io.ReadAll(speech.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read speech audio: %w", err)
	}
	if len(audio) == 0 {
		return nil, nil, fmt.Errorf("speech returned empty audio")
	}
	audio, packets, err := opusAudioAndPackets(audio)
	if err != nil {
		return nil, nil, err
	}
	return audio, packets, nil
}

func (d *personaDriver) transcribe(ctx context.Context, audioPath string) (string, error) {
	if d.transcribeAudioFile != nil {
		return d.transcribeAudioFile(ctx, audioPath)
	}
	file, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("open audio for asr: %w", err)
	}
	defer file.Close()
	transcription, err := d.client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		File:  file,
		Model: openai.AudioModel(d.cfg.Models.ASR),
	})
	if err != nil {
		return "", fmt.Errorf("transcription: %w", err)
	}
	text := strings.TrimSpace(transcription.Text)
	if text == "" {
		return "", fmt.Errorf("transcription returned empty text")
	}
	return text, nil
}

func (d *personaDriver) writeRoundAudio(index int, audio []byte) (string, error) {
	return d.writeRoundAudioFile(index, "input", audio)
}

func (d *personaDriver) writeRoundAudioFile(index int, name string, audio []byte) (string, error) {
	if d.cfg.OutputDir == "" {
		dir, err := os.MkdirTemp("", "gizclaw-workspacetest-*")
		if err != nil {
			return "", fmt.Errorf("create output dir: %w", err)
		}
		d.cfg.OutputDir = dir
	}
	if err := os.MkdirAll(d.cfg.OutputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	path := filepath.Join(d.cfg.OutputDir, fmt.Sprintf("round-%02d-%s.ogg", index, name))
	if err := os.WriteFile(path, audio, 0o644); err != nil {
		return "", fmt.Errorf("write round audio: %w", err)
	}
	return path, nil
}

func opusPacketsFromOgg(audio []byte) ([][]byte, error) {
	var packets [][]byte
	for packet, err := range ogg.Packets(bytes.NewReader(audio)) {
		if err != nil {
			return nil, fmt.Errorf("read ogg opus packets: %w", err)
		}
		if len(packet.Data) == 0 {
			continue
		}
		if bytes.HasPrefix(packet.Data, []byte("OpusHead")) || bytes.HasPrefix(packet.Data, []byte("OpusTags")) {
			continue
		}
		packets = append(packets, append([]byte(nil), packet.Data...))
	}
	if len(packets) == 0 {
		return nil, fmt.Errorf("no opus payload packets found")
	}
	return packets, nil
}

func opusAudioAndPackets(audio []byte) ([]byte, [][]byte, error) {
	switch {
	case bytes.HasPrefix(audio, []byte("OggS")):
		packets, err := opusPacketsFromOgg(audio)
		if err != nil {
			return nil, nil, err
		}
		return audio, packets, nil
	case isMP3Audio(audio):
		packets, err := opusPacketsFromMP3(audio, 16000, 1)
		if err != nil {
			return nil, nil, err
		}
		oggAudio, err := oggOpusFromPackets(16000, 1, packets)
		if err != nil {
			return nil, nil, fmt.Errorf("wrap mp3-derived opus packets: %w", err)
		}
		return oggAudio, packets, nil
	default:
		return nil, nil, fmt.Errorf("unsupported speech audio container")
	}
}

func opusPacketsFromMP3(audio []byte, targetSampleRate, targetChannels int) ([][]byte, error) {
	decoded, sampleRate, channels, err := mp3.DecodeFull(bytes.NewReader(audio))
	if err != nil {
		return nil, fmt.Errorf("decode mp3 speech audio: %w", err)
	}
	if sampleRate <= 0 {
		return nil, fmt.Errorf("decode mp3 speech audio: invalid sample rate %d", sampleRate)
	}
	if channels != 1 && channels != 2 {
		return nil, fmt.Errorf("decode mp3 speech audio: unsupported channels %d", channels)
	}
	srcFmt := resampler.Format{SampleRate: sampleRate, Stereo: channels == 2}
	dstFmt := resampler.Format{SampleRate: targetSampleRate, Stereo: targetChannels == 2}
	pcm := decoded
	if srcFmt != dstFmt {
		rs, err := resampler.New(bytes.NewReader(decoded), srcFmt, dstFmt)
		if err != nil {
			return nil, fmt.Errorf("create mp3 speech resampler: %w", err)
		}
		defer func() {
			_ = rs.Close()
		}()
		pcm, err = io.ReadAll(rs)
		if err != nil {
			return nil, fmt.Errorf("resample mp3 speech pcm: %w", err)
		}
	}

	if len(pcm)%2 != 0 {
		return nil, fmt.Errorf("mp3 speech pcm length must be even, got %d", len(pcm))
	}
	samples := make([]int16, len(pcm)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("mp3 speech decoded to empty pcm")
	}

	enc, err := opus.NewEncoder(targetSampleRate, targetChannels, opus.ApplicationAudio)
	if err != nil {
		return nil, fmt.Errorf("create speech opus encoder: %w", err)
	}
	defer func() {
		_ = enc.Close()
	}()

	frameSize := targetSampleRate / 50
	samplesPerFrame := frameSize * targetChannels
	packets := make([][]byte, 0, (len(samples)+samplesPerFrame-1)/samplesPerFrame)
	for offset := 0; offset < len(samples); offset += samplesPerFrame {
		frame := make([]int16, samplesPerFrame)
		copy(frame, samples[offset:min(offset+samplesPerFrame, len(samples))])
		packet, err := enc.Encode(frame, frameSize)
		if err != nil {
			return nil, fmt.Errorf("encode speech opus frame: %w", err)
		}
		packets = append(packets, packet)
	}
	if len(packets) == 0 {
		return nil, fmt.Errorf("mp3 speech produced no opus packets")
	}
	return packets, nil
}

func isMP3Audio(audio []byte) bool {
	return len(audio) >= 3 && string(audio[:3]) == "ID3" || len(audio) >= 2 && audio[0] == 0xff && audio[1]&0xe0 == 0xe0
}

func oggOpusFromPackets(sampleRate, channels int, packets [][]byte) ([]byte, error) {
	var out bytes.Buffer
	writer, err := ogg.NewStreamWriter(&out, 456)
	if err != nil {
		return nil, err
	}
	header := opusHeadPacket(sampleRate, channels)
	tags := opusTagsPacket("workspacetest")
	if _, err := writer.WritePacket(header, 0, false); err != nil {
		return nil, err
	}
	if _, err := writer.WritePacket(tags, 0, false); err != nil {
		return nil, err
	}
	for i, packet := range packets {
		if len(packet) == 0 {
			continue
		}
		if _, err := writer.WritePacket(packet, uint64(i+1), i == len(packets)-1); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

func opusHeadPacket(sampleRate, channels int) []byte {
	packet := make([]byte, 19)
	copy(packet[:8], "OpusHead")
	packet[8] = 1
	packet[9] = byte(channels)
	binary.LittleEndian.PutUint32(packet[12:16], uint32(sampleRate))
	return packet
}

func opusTagsPacket(vendor string) []byte {
	vendorBytes := []byte(vendor)
	packet := make([]byte, 8+4+len(vendorBytes)+4)
	copy(packet[:8], "OpusTags")
	binary.LittleEndian.PutUint32(packet[8:12], uint32(len(vendorBytes)))
	copy(packet[12:12+len(vendorBytes)], vendorBytes)
	return packet
}

func cleanUtterance(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "`\"'“”‘’ \t\r\n")
	text = strings.ReplaceAll(text, "\n", "")
	return strings.TrimSpace(text)
}

func eventLabel(event apitypes.PeerStreamEvent) string {
	if event.Label == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(*event.Label))
}

func assertTextSimilar(name, expected, actual string, minRatio float64) error {
	expectedNorm := normalizeTranscript(expected)
	actualNorm := normalizeTranscript(actual)
	if expectedNorm == "" || actualNorm == "" {
		return fmt.Errorf("%s empty after normalization: expected=%q actual=%q", name, expected, actual)
	}
	ratio := lcsRatio(expectedNorm, actualNorm)
	if ratio >= minRatio {
		return nil
	}
	return fmt.Errorf("%s mismatch: similarity %.2f below %.2f: expected %q normalized %q, got %q normalized %q", name, ratio, minRatio, expected, expectedNorm, actual, actualNorm)
}

func isAgentAlreadyRunning(err error) bool {
	return err != nil && strings.Contains(err.Error(), "workspace already has a running agent")
}

func mergeTranscriptText(current, chunk string) string {
	current = strings.TrimSpace(current)
	chunk = strings.TrimSpace(chunk)
	if chunk == "" {
		return current
	}
	if current == "" {
		return chunk
	}
	currentNorm := normalizeTranscript(current)
	chunkNorm := normalizeTranscript(chunk)
	if chunkNorm == "" {
		return current
	}
	if currentNorm == "" {
		return chunk
	}
	if strings.Contains(currentNorm, chunkNorm) {
		if transcriptTextScore(chunk) > transcriptTextScore(current) {
			return chunk
		}
		return current
	}
	if strings.Contains(chunkNorm, currentNorm) {
		return chunk
	}
	if lcsRatio(currentNorm, chunkNorm) >= 0.80 {
		if transcriptTextScore(chunk) >= transcriptTextScore(current) {
			return chunk
		}
		return current
	}
	if suffix := normalizedSuffixAfterPrefix(currentNorm, chunk); suffix != "" {
		return current + suffix
	}
	return current + chunk
}

func transcriptTextScore(text string) int {
	score := runeCount(text)
	for _, r := range text {
		switch r {
		case '，', '。', '？', '！', ',', '.', '?', '!':
			score += 2
		}
	}
	return score
}

func normalizedSuffixAfterPrefix(prefixNorm, text string) string {
	if prefixNorm == "" {
		return text
	}
	matched := 0
	for i, r := range text {
		norm := normalizeTranscript(string(r))
		if norm == "" {
			continue
		}
		if matched >= len(prefixNorm) || !strings.HasPrefix(prefixNorm[matched:], norm) {
			return ""
		}
		matched += len(norm)
		if matched == len(prefixNorm) {
			return text[i+len(string(r)):]
		}
	}
	return ""
}

func lcsRatio(a, b string) float64 {
	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 || len(br) == 0 {
		return 0
	}
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for i := range ar {
		for j := range br {
			if ar[i] == br[j] {
				curr[j+1] = prev[j] + 1
			} else if curr[j] > prev[j+1] {
				curr[j+1] = curr[j]
			} else {
				curr[j+1] = prev[j+1]
			}
		}
		prev, curr = curr, prev
		clear(curr)
	}
	return float64(prev[len(br)]) / float64(max(len(ar), len(br)))
}

func normalizeTranscript(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= '\u4e00' && r <= '\u9fff') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func encodeJSONLine(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(data)
}

func runeCount(s string) int {
	return utf8.RuneCountInString(s)
}

type chatTransport struct {
	client         *gizcli.Client
	stream         peerChunkStream
	packetInterval time.Duration
	events         chan timedPeerEvent
	opusPackets    <-chan timedPeerPacket
	errs           chan error
}

type peerChunkStream interface {
	genx.Stream
	Push(context.Context, *genx.MessageChunk) error
}

type timedPeerEvent struct {
	event      apitypes.PeerStreamEvent
	receivedAt time.Time
}

func newTimedPeerEvent(event apitypes.PeerStreamEvent) timedPeerEvent {
	return timedPeerEvent{event: event, receivedAt: time.Now()}
}

func (e timedPeerEvent) since(start time.Time) time.Duration {
	if e.receivedAt.IsZero() {
		return time.Since(start)
	}
	return e.receivedAt.Sub(start)
}

type timedPeerPacket struct {
	frame      []byte
	receivedAt time.Time
}

func newTimedPeerPacket(frame []byte) timedPeerPacket {
	return timedPeerPacket{frame: append([]byte(nil), frame...), receivedAt: time.Now()}
}

func (p timedPeerPacket) since(start time.Time) time.Duration {
	if p.receivedAt.IsZero() {
		return time.Since(start)
	}
	return p.receivedAt.Sub(start)
}

func afterStartLatency(receivedAt, start time.Time) time.Duration {
	latency, _ := afterStartLatencyStatus(receivedAt, start)
	return latency
}

func afterStartLatencyStatus(receivedAt, start time.Time) (time.Duration, bool) {
	if receivedAt.IsZero() {
		return time.Since(start), false
	}
	latency := receivedAt.Sub(start)
	if latency <= 0 {
		return time.Nanosecond, true
	}
	return latency, false
}

func newChatTransport(client *gizcli.Client) (*chatTransport, error) {
	stream, err := client.OpenPeerStream(128)
	if err != nil {
		return nil, err
	}
	packets := make(chan timedPeerPacket, 128)
	t := &chatTransport{
		client: client,
		stream: stream,
		// Realtime dialogue backends expect microphone-like audio cadence.
		packetInterval: 20 * time.Millisecond,
		events:         make(chan timedPeerEvent, 128),
		opusPackets:    packets,
		errs:           make(chan error, 1),
	}
	go t.readChunks(packets)
	return t, nil
}

func (t *chatTransport) close() {
	if t == nil {
		return
	}
	if t.stream != nil {
		_ = t.stream.Close()
	}
}

func (t *chatTransport) drain() {
	if t == nil {
		return
	}
	for {
		select {
		case <-t.events:
		case <-t.opusPackets:
		default:
			return
		}
	}
}

func (t *chatTransport) readChunks(packets chan<- timedPeerPacket) {
	for {
		chunk, err := t.stream.Next()
		if err != nil {
			if err == io.EOF {
				return
			}
			select {
			case t.errs <- err:
			default:
			}
			return
		}
		if blob, ok := chunk.Part.(*genx.Blob); ok && strings.EqualFold(strings.TrimSpace(blob.MIMEType), "audio/opus") && len(blob.Data) > 0 {
			select {
			case packets <- newTimedPeerPacket(blob.Data):
			default:
			}
			continue
		}
		for _, event := range transportEventsFromChunk(chunk) {
			select {
			case t.events <- newTimedPeerEvent(event):
			default:
			}
		}
	}
}

func (t *chatTransport) sendAudioTurn(ctx context.Context, streamID string, packets [][]byte) error {
	label := "workspacetest"
	if err := t.stream.Push(ctx, &genx.MessageChunk{
		Ctrl: &genx.StreamCtrl{StreamID: streamID, Label: label, BeginOfStream: true},
	}); err != nil {
		return err
	}
	timestamp := time.Now().UnixMilli()
	for i, packet := range packets {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := t.stream.Push(ctx, &genx.MessageChunk{
			Part: &genx.Blob{MIMEType: "audio/opus", Data: packet},
			Ctrl: &genx.StreamCtrl{StreamID: streamID, Label: label, Timestamp: timestamp},
		}); err != nil {
			return err
		}
		timestamp += 20
		if t.packetInterval > 0 && i+1 < len(packets) {
			timer := time.NewTimer(t.packetInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return t.stream.Push(ctx, &genx.MessageChunk{
		Ctrl: &genx.StreamCtrl{StreamID: streamID, Label: label, EndOfStream: true},
	})
}

func transportEventsFromChunk(chunk *genx.MessageChunk) []apitypes.PeerStreamEvent {
	if chunk == nil {
		return nil
	}
	var out []apitypes.PeerStreamEvent
	if chunk.IsBeginOfStream() {
		out = append(out, transportEventFromChunk(chunk, apitypes.PeerStreamEventTypeBos, nil))
	}
	if text, ok := chunk.Part.(genx.Text); ok {
		value := string(text)
		eventType := apitypes.PeerStreamEventTypeTextDelta
		if chunk.IsEndOfStream() {
			eventType = apitypes.PeerStreamEventTypeTextDone
		}
		out = append(out, transportEventFromChunk(chunk, eventType, &value))
		return out
	}
	if chunk.IsEndOfStream() {
		out = append(out, transportEventFromChunk(chunk, apitypes.PeerStreamEventTypeEos, nil))
	}
	return out
}

func transportEventFromChunk(chunk *genx.MessageChunk, eventType apitypes.PeerStreamEventType, text *string) apitypes.PeerStreamEvent {
	event := apitypes.PeerStreamEvent{V: 1, Type: eventType, Text: text}
	if chunk == nil || chunk.Ctrl == nil {
		return event
	}
	if chunk.Ctrl.StreamID != "" {
		event.StreamId = &chunk.Ctrl.StreamID
	}
	if chunk.Ctrl.Label != "" {
		event.Label = &chunk.Ctrl.Label
	}
	if chunk.Ctrl.Timestamp != 0 {
		event.Timestamp = &chunk.Ctrl.Timestamp
	}
	return event
}
