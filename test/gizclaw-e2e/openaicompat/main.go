package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

type opStats struct {
	Name      string
	Total     time.Duration
	First     time.Duration
	FirstName string
	Events    int
	Bytes     int
	Chars     int
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "openai-compat: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := loadConfig(args)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	httpClient := &http.Client{Timeout: cfg.Timeout}
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(cfg.BaseURL),
		option.WithHTTPClient(httpClient),
	)

	var stats []opStats
	fail := func(err error) error {
		if len(stats) > 0 {
			printStats(stats)
		}
		return err
	}
	completion, stat, err := runChat(ctx, client, cfg.ModelID)
	stats = append(stats, stat)
	if err != nil {
		return fail(err)
	}
	streamingCompletion, stat, err := runStreamingChat(ctx, client, cfg.ModelID)
	stats = append(stats, stat)
	if err != nil {
		return fail(err)
	}

	voiceID := cfg.VoiceID
	if voiceID == "" {
		voiceID, err = firstVoiceID(ctx, httpClient, cfg)
		if err != nil {
			return fail(err)
		}
	}

	speechPath, speechBytes, stat, err := runSpeech(ctx, client, cfg.OutputDir, cfg.TTSModelID, voiceID, "speech.mp3", completion)
	stats = append(stats, stat)
	if err != nil {
		return fail(err)
	}
	streamPath, streamBytes, stat, err := runSpeechStream(ctx, client, cfg.OutputDir, cfg.TTSModelID, voiceID, "speech-stream.mp3", streamingCompletion)
	stats = append(stats, stat)
	if err != nil {
		return fail(err)
	}
	var transcription string
	var streamingTranscription string
	if cfg.ASRModelID != "" {
		transcription, stat, err = runTranscription(ctx, client, cfg.ASRModelID, speechPath)
		stats = append(stats, stat)
		if err != nil {
			return fail(err)
		}
		if err := assertTranscriptionMatches("non-stream", completion, transcription); err != nil {
			return fail(err)
		}
		streamingTranscription, stat, err = runTranscriptionStream(ctx, client, cfg.ASRModelID, streamPath)
		stats = append(stats, stat)
		if err != nil {
			return fail(err)
		}
		if err := assertTranscriptionSimilar("stream", streamingCompletion, streamingTranscription, 0.85); err != nil {
			return fail(err)
		}
	}

	fmt.Printf("base_url=%s\n", cfg.BaseURL)
	fmt.Printf("model=%s\n", cfg.ModelID)
	fmt.Printf("tts_model=%s\n", cfg.TTSModelID)
	fmt.Printf("voice=%s\n", voiceID)
	fmt.Printf("chat=%q\n", strings.TrimSpace(completion))
	fmt.Printf("chat_stream=%q\n", strings.TrimSpace(streamingCompletion))
	fmt.Printf("speech=%s bytes=%d\n", speechPath, speechBytes)
	fmt.Printf("speech_stream=%s bytes=%d\n", streamPath, streamBytes)
	if cfg.ASRModelID != "" {
		fmt.Printf("asr_model=%s\n", cfg.ASRModelID)
		fmt.Printf("transcription=%q\n", strings.TrimSpace(transcription))
		fmt.Printf("transcription_stream=%q\n", strings.TrimSpace(streamingTranscription))
	}
	printStats(stats)
	return nil
}

func runChat(ctx context.Context, client openai.Client, modelID string) (string, opStats, error) {
	start := time.Now()
	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(modelID),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a smoke test endpoint. Follow the user instruction exactly."),
			openai.UserMessage("Reply with this exact Chinese sentence only, no punctuation: 小猫今天开心跑步"),
		},
	})
	stat := opStats{Name: "chat", Total: time.Since(start)}
	if err != nil {
		return "", stat, fmt.Errorf("chat completion with model %q: %w", modelID, err)
	}
	if len(completion.Choices) == 0 {
		return "", stat, fmt.Errorf("chat completion with model %q returned no choices", modelID)
	}
	text := strings.TrimSpace(completion.Choices[0].Message.Content)
	if text == "" {
		return "", stat, fmt.Errorf("chat completion with model %q returned empty content", modelID)
	}
	stat.Chars = utf8.RuneCountInString(text)
	return text, stat, nil
}

func runStreamingChat(ctx context.Context, client openai.Client, modelID string) (string, opStats, error) {
	start := time.Now()
	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(modelID),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a smoke test endpoint. Write concise Simplified Chinese without markdown."),
			openai.UserMessage("讲一个适合语音播放的小故事，120 到 180 个汉字，只有正文，不要标题。"),
		},
	})
	defer stream.Close()

	var completion strings.Builder
	stat := opStats{Name: "chat_stream", FirstName: "first_token"}
	for stream.Next() {
		stat.Events++
		chunk := stream.Current()
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta != "" && stat.First == 0 {
			stat.First = time.Since(start)
		}
		completion.WriteString(delta)
	}
	stat.Total = time.Since(start)
	if err := stream.Err(); err != nil {
		return "", stat, fmt.Errorf("streaming chat completion with model %q: %w", modelID, err)
	}
	if strings.TrimSpace(completion.String()) == "" {
		return "", stat, fmt.Errorf("streaming chat completion with model %q returned no content", modelID)
	}
	stat.Chars = utf8.RuneCountInString(completion.String())
	return completion.String(), stat, nil
}

func runSpeech(ctx context.Context, client openai.Client, outputDir, modelID, voiceID, filename, input string) (string, int, opStats, error) {
	start := time.Now()
	speech, err := client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Input:          input,
		Model:          openai.SpeechModel(modelID),
		Voice:          openai.AudioSpeechNewParamsVoice(voiceID),
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
	})
	stat := opStats{Name: "speech", FirstName: "first_byte"}
	if err != nil {
		stat.Total = time.Since(start)
		return "", 0, stat, fmt.Errorf("speech with model %q voice %q: %w", modelID, voiceID, err)
	}
	defer speech.Body.Close()

	audio, firstByte, err := readAllMeasured(speech.Body, start)
	stat.Total = time.Since(start)
	stat.First = firstByte
	stat.Bytes = len(audio)
	if err != nil {
		return "", 0, stat, fmt.Errorf("read speech audio: %w", err)
	}
	if len(audio) == 0 {
		return "", 0, stat, fmt.Errorf("speech with model %q voice %q returned empty audio", modelID, voiceID)
	}
	path := filepath.Join(outputDir, filename)
	if err := os.WriteFile(path, audio, 0o644); err != nil {
		return "", 0, stat, fmt.Errorf("write speech audio: %w", err)
	}
	return path, len(audio), stat, nil
}

func runSpeechStream(ctx context.Context, client openai.Client, outputDir, modelID, voiceID, filename, input string) (string, int, opStats, error) {
	start := time.Now()
	speech, err := client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Input:          input,
		Model:          openai.SpeechModel(modelID),
		Voice:          openai.AudioSpeechNewParamsVoice(voiceID),
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
		StreamFormat:   openai.AudioSpeechNewParamsStreamFormatSSE,
	})
	stat := opStats{Name: "speech_stream", FirstName: "first_audio_delta"}
	if err != nil {
		stat.Total = time.Since(start)
		return "", 0, stat, fmt.Errorf("streaming speech with model %q voice %q: %w", modelID, voiceID, err)
	}
	defer speech.Body.Close()
	if got := speech.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		stat.Total = time.Since(start)
		return "", 0, stat, fmt.Errorf("streaming speech content type = %q, want text/event-stream", got)
	}
	audio, events, firstDelta, err := readSpeechSSE(speech.Body, start)
	stat.Total = time.Since(start)
	stat.First = firstDelta
	stat.Events = events
	stat.Bytes = len(audio)
	if err != nil {
		return "", 0, stat, err
	}
	path := filepath.Join(outputDir, filename)
	if err := os.WriteFile(path, audio, 0o644); err != nil {
		return "", 0, stat, fmt.Errorf("write streamed speech audio: %w", err)
	}
	return path, len(audio), stat, nil
}

func runTranscription(ctx context.Context, client openai.Client, modelID, audioPath string) (string, opStats, error) {
	file, err := os.Open(audioPath)
	if err != nil {
		return "", opStats{Name: "transcription"}, fmt.Errorf("open transcription audio: %w", err)
	}
	defer file.Close()

	start := time.Now()
	transcription, err := client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		File:  file,
		Model: openai.AudioModel(modelID),
	})
	stat := opStats{Name: "transcription", Total: time.Since(start)}
	if err != nil {
		return "", stat, fmt.Errorf("transcription with model %q: %w", modelID, err)
	}
	if strings.TrimSpace(transcription.Text) == "" {
		return "", stat, fmt.Errorf("transcription with model %q returned empty text", modelID)
	}
	stat.Chars = utf8.RuneCountInString(transcription.Text)
	return transcription.Text, stat, nil
}

func runTranscriptionStream(ctx context.Context, client openai.Client, modelID, audioPath string) (string, opStats, error) {
	file, err := os.Open(audioPath)
	if err != nil {
		return "", opStats{Name: "transcription_stream"}, fmt.Errorf("open streaming transcription audio: %w", err)
	}
	defer file.Close()

	start := time.Now()
	stream := client.Audio.Transcriptions.NewStreaming(ctx, openai.AudioTranscriptionNewParams{
		File:  file,
		Model: openai.AudioModel(modelID),
	})
	defer stream.Close()

	var delta strings.Builder
	var doneText string
	stat := opStats{Name: "transcription_stream", FirstName: "first_delta"}
	for stream.Next() {
		stat.Events++
		event := stream.Current()
		switch event.Type {
		case "transcript.text.delta":
			if event.Delta != "" && stat.First == 0 {
				stat.First = time.Since(start)
			}
			delta.WriteString(event.Delta)
		case "transcript.text.done":
			doneText = event.Text
		default:
			stat.Total = time.Since(start)
			return "", stat, fmt.Errorf("unexpected transcription stream event type %q", event.Type)
		}
	}
	stat.Total = time.Since(start)
	if err := stream.Err(); err != nil {
		return "", stat, fmt.Errorf("streaming transcription with model %q: %w", modelID, err)
	}
	if strings.TrimSpace(doneText) != "" {
		stat.Chars = utf8.RuneCountInString(doneText)
		return doneText, stat, nil
	}
	text := delta.String()
	if strings.TrimSpace(text) == "" {
		return "", stat, fmt.Errorf("streaming transcription with model %q returned empty text", modelID)
	}
	stat.Chars = utf8.RuneCountInString(text)
	return text, stat, nil
}

func assertTranscriptionMatches(name, expected, actual string) error {
	expectedNorm := normalizeTranscript(expected)
	actualNorm := normalizeTranscript(actual)
	if expectedNorm == "" {
		return fmt.Errorf("%s transcription expected text is empty after normalization", name)
	}
	if actualNorm == "" {
		return fmt.Errorf("%s transcription actual text is empty after normalization", name)
	}
	if expectedNorm == actualNorm {
		return nil
	}
	return fmt.Errorf("%s transcription mismatch: expected %q normalized %q, got %q normalized %q", name, expected, expectedNorm, actual, actualNorm)
}

func assertTranscriptionSimilar(name, expected, actual string, minRatio float64) error {
	expectedNorm := normalizeTranscript(expected)
	actualNorm := normalizeTranscript(actual)
	if expectedNorm == "" {
		return fmt.Errorf("%s transcription expected text is empty after normalization", name)
	}
	if actualNorm == "" {
		return fmt.Errorf("%s transcription actual text is empty after normalization", name)
	}
	ratio := lcsRatio(expectedNorm, actualNorm)
	if ratio >= minRatio {
		return nil
	}
	return fmt.Errorf("%s transcription mismatch: similarity %.2f below %.2f: expected %q normalized %q, got %q normalized %q", name, ratio, minRatio, expected, expectedNorm, actual, actualNorm)
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

func readSpeechSSE(r io.Reader, start time.Time) ([]byte, int, time.Duration, error) {
	var audio []byte
	var deltaCount int
	var firstDelta time.Duration
	var done bool
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		var event struct {
			Audio string `json:"audio"`
			Done  bool   `json:"done"`
			Type  string `json:"type"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &event); err != nil {
			return nil, deltaCount, firstDelta, fmt.Errorf("decode streaming speech event %q: %w", line, err)
		}
		switch event.Type {
		case "speech.audio.delta":
			if firstDelta == 0 {
				firstDelta = time.Since(start)
			}
			chunk, err := base64.StdEncoding.DecodeString(event.Audio)
			if err != nil {
				return nil, deltaCount, firstDelta, fmt.Errorf("decode streaming speech audio delta: %w", err)
			}
			if len(chunk) == 0 {
				return nil, deltaCount, firstDelta, fmt.Errorf("streaming speech audio delta is empty")
			}
			audio = append(audio, chunk...)
			deltaCount++
		case "speech.audio.done":
			done = true
		default:
			return nil, deltaCount, firstDelta, fmt.Errorf("unexpected streaming speech event type %q", event.Type)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, deltaCount, firstDelta, fmt.Errorf("read streaming speech events: %w", err)
	}
	if deltaCount == 0 {
		return nil, deltaCount, firstDelta, fmt.Errorf("streaming speech returned no audio delta events")
	}
	if !done {
		return nil, deltaCount, firstDelta, fmt.Errorf("streaming speech did not return speech.audio.done")
	}
	if len(audio) == 0 {
		return nil, deltaCount, firstDelta, fmt.Errorf("streaming speech decoded audio is empty")
	}
	return audio, deltaCount, firstDelta, nil
}

func readAllMeasured(r io.Reader, start time.Time) ([]byte, time.Duration, error) {
	var out []byte
	var firstByte time.Duration
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if firstByte == 0 {
				firstByte = time.Since(start)
			}
			out = append(out, buf[:n]...)
		}
		if err == io.EOF {
			return out, firstByte, nil
		}
		if err != nil {
			return out, firstByte, err
		}
	}
}

func printStats(stats []opStats) {
	fmt.Println("stats:")
	var nonStream []opStats
	var stream []opStats
	for _, stat := range stats {
		if strings.HasSuffix(stat.Name, "_stream") {
			stream = append(stream, stat)
			continue
		}
		nonStream = append(nonStream, stat)
	}
	if len(nonStream) > 0 {
		fmt.Println("non-stream:")
		printStatsTable(nonStream)
	}
	if len(stream) > 0 {
		fmt.Println("stream:")
		printStatsTable(stream)
	}
}

func printStatsTable(stats []opStats) {
	headers := []string{"operation", "total", "first", "events", "bytes", "chars"}
	rows := make([][]string, 0, len(stats))
	for _, stat := range stats {
		first := "-"
		if stat.FirstName != "" && stat.First > 0 {
			first = stat.First.Round(time.Millisecond).String()
		}
		rows = append(rows, []string{
			stat.Name,
			stat.Total.Round(time.Millisecond).String(),
			first,
			formatCount(stat.Events),
			formatCount(stat.Bytes),
			formatCount(stat.Chars),
		})
	}
	printTable(headers, rows)
}

func printTable(headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = utf8.RuneCountInString(header)
	}
	for _, row := range rows {
		for i, value := range row {
			if w := utf8.RuneCountInString(value); w > widths[i] {
				widths[i] = w
			}
		}
	}

	printTableBorder("┌", "┬", "┐", widths)
	printTableRow(headers, widths)
	printTableBorder("├", "┼", "┤", widths)
	for _, row := range rows {
		printTableRow(row, widths)
	}
	printTableBorder("└", "┴", "┘", widths)
}

func printTableBorder(left, mid, right string, widths []int) {
	fmt.Print(left)
	for i, width := range widths {
		if i > 0 {
			fmt.Print(mid)
		}
		fmt.Print(strings.Repeat("─", width+2))
	}
	fmt.Println(right)
}

func printTableRow(row []string, widths []int) {
	fmt.Print("│")
	for i, width := range widths {
		value := ""
		if i < len(row) {
			value = row[i]
		}
		fmt.Printf(" %s%s │", value, strings.Repeat(" ", width-utf8.RuneCountInString(value)))
	}
	fmt.Println()
}

func formatCount(n int) string {
	if n <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d", n)
}

func firstVoiceID(ctx context.Context, client *http.Client, cfg config) (string, error) {
	voicesURL, err := url.Parse(cfg.BaseURL + "/voices")
	if err != nil {
		return "", fmt.Errorf("parse voices url: %w", err)
	}
	q := voicesURL.Query()
	q.Set("providerKind", "volc-tenant")
	q.Set("limit", "20")
	voicesURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, voicesURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create voices request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("list voices through %s: %w", voicesURL.String(), err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read voices response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("list voices status = %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Data []struct {
			ID       string `json:"id"`
			Provider struct {
				Kind string `json:"kind"`
			} `json:"provider"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decode voices response: %w", err)
	}
	for _, voice := range parsed.Data {
		if voice.ID != "" && voice.Provider.Kind == "volc-tenant" {
			return voice.ID, nil
		}
	}
	return "", fmt.Errorf("no volc voice returned by %s: %.512s", voicesURL.String(), string(body))
}
