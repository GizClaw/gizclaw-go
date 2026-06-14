package transformers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"strings"

	"github.com/GizClaw/doubao-speech-go"
	"github.com/GizClaw/gizclaw-go/pkg/genx"
)

// DoubaoASRSAUC is an ASR transformer using Doubao BigModel ASR (大模型语音识别).
//
// Resource ID: volc.bigasr.sauc.duration
//
// Input type: audio/* (audio/ogg, audio/pcm, etc.)
// Output type: text/plain
//
// EoS Handling:
//   - When receiving an audio/* EoS marker, finish current ASR, emit results, then emit text/plain EoS
//   - Non-audio chunks are passed through unchanged
//
// Note: The input audio format must match the configured format.
type DoubaoASRSAUC struct {
	client     *doubaospeech.Client
	format     string
	sampleRate int
	channels   int
	bits       int
	language   string
	enableITN  bool
	enablePunc bool
	hotwords   []string
	resultType string // "single" (default) or "full"
	resourceID string
	chunkSize  int

	newSession func(context.Context) (doubaoASRSession, error)
}

var _ genx.Transformer = (*DoubaoASRSAUC)(nil)

type doubaoASRSession interface {
	SendAudio(context.Context, []byte, bool) error
	Recv() iter.Seq2[*doubaospeech.ASRV2Result, error]
	Close() error
}

// DoubaoASRSAUCOption is a functional option for DoubaoASRSAUC.
type DoubaoASRSAUCOption func(*DoubaoASRSAUC)

// WithDoubaoASRSAUCFormat sets the audio format (pcm, wav, mp3, ogg_opus).
func WithDoubaoASRSAUCFormat(format string) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.format = format
	}
}

// WithDoubaoASRSAUCSampleRate sets the sample rate (8000, 16000, etc.).
func WithDoubaoASRSAUCSampleRate(sampleRate int) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.sampleRate = sampleRate
	}
}

// WithDoubaoASRSAUCChannels sets the number of channels (1 or 2).
func WithDoubaoASRSAUCChannels(channels int) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.channels = channels
	}
}

// WithDoubaoASRSAUCBits sets the bits per sample (16, etc.).
func WithDoubaoASRSAUCBits(bits int) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.bits = bits
	}
}

// WithDoubaoASRSAUCLanguage sets the language (zh-CN, en-US, ja-JP, etc.).
func WithDoubaoASRSAUCLanguage(language string) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.language = language
	}
}

// WithDoubaoASRSAUCEnableITN enables Inverse Text Normalization.
func WithDoubaoASRSAUCEnableITN(enable bool) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.enableITN = enable
	}
}

// WithDoubaoASRSAUCEnablePunc enables punctuation.
func WithDoubaoASRSAUCEnablePunc(enable bool) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.enablePunc = enable
	}
}

// WithDoubaoASRSAUCHotwords sets hotwords for recognition boost.
func WithDoubaoASRSAUCHotwords(hotwords []string) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.hotwords = hotwords
	}
}

// WithDoubaoASRSAUCResultType sets the result type.
// Options: "single" (default, only definite results), "full" (all results including interim).
func WithDoubaoASRSAUCResultType(resultType string) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.resultType = resultType
	}
}

// WithDoubaoASRSAUCResourceID sets the ASR resource ID.
func WithDoubaoASRSAUCResourceID(resourceID string) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.resourceID = resourceID
	}
}

// WithDoubaoASRSAUCChunkSize sets the maximum audio frame size sent to Doubao.
func WithDoubaoASRSAUCChunkSize(chunkSize int) DoubaoASRSAUCOption {
	return func(t *DoubaoASRSAUC) {
		t.chunkSize = chunkSize
	}
}

// NewDoubaoASRSAUC creates a new DoubaoASRSAUC transformer.
//
// Parameters:
//   - client: Doubao speech client
//   - opts: Optional configuration
func NewDoubaoASRSAUC(client *doubaospeech.Client, opts ...DoubaoASRSAUCOption) *DoubaoASRSAUC {
	t := &DoubaoASRSAUC{
		client:     client,
		format:     string(doubaospeech.FormatOGG),
		sampleRate: 16000,
		channels:   1,
		bits:       16,
		language:   "zh-CN",
		enableITN:  true,
		enablePunc: true,
		resultType: "single", // only definite results
		resourceID: doubaospeech.ResourceASRStream,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// DoubaoASRSAUCCtxKey is the context key for runtime options.
type doubaoASRSAUCCtxKey struct{}

// DoubaoASRSAUCCtxOptions are runtime options passed via context.
// TODO: Add fields as needed for runtime configuration.
type DoubaoASRSAUCCtxOptions struct{}

// WithDoubaoASRSAUCCtxOptions attaches runtime options to context.
func WithDoubaoASRSAUCCtxOptions(ctx context.Context, opts DoubaoASRSAUCCtxOptions) context.Context {
	return context.WithValue(ctx, doubaoASRSAUCCtxKey{}, opts)
}

// Transform converts audio Blob chunks to Text chunks.
// DoubaoASRSAUC creates sessions on demand, so it returns immediately.
// The ctx is unused (session creation happens lazily in the loop);
// the goroutine lifetime is governed by the input Stream.
func (t *DoubaoASRSAUC) Transform(_ context.Context, _ string, input genx.Stream) (genx.Stream, error) {
	output := newBufferStream(100)

	go t.transformLoop(input, output)

	return output, nil
}

func (t *DoubaoASRSAUC) transformLoop(input genx.Stream, output *bufferStream) {
	defer output.Close()

	// Local cancel context tied to the loop lifecycle.
	// When the loop exits, defer cancel() cancels any in-flight WebSocket
	// dial or audio send operation.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Track last chunk for metadata
	var lastChunk *genx.MessageChunk
	var session doubaoASRSession
	var resultsCh chan *genx.MessageChunk
	var resultsDone chan error
	var resultsForwarded chan struct{}
	var pendingAudio []byte

	// Helper to start a new ASR session
	startSession := func() error {
		var err error
		openSession := t.openSession
		if t.newSession != nil {
			openSession = t.newSession
		}
		session, err = openSession(ctx)
		if err != nil {
			return err
		}
		resultsCh = make(chan *genx.MessageChunk, 100)
		resultsDone = make(chan error, 1)
		resultsForwarded = make(chan struct{})
		go t.receiveResults(session, lastChunk, resultsCh, resultsDone)
		// Forward results to output as they arrive
		go func() {
			defer close(resultsForwarded)
			for chunk := range resultsCh {
				output.Push(chunk)
			}
		}()
		return nil
	}

	// Helper to finish current session
	finishSession := func() error {
		if session == nil {
			return nil
		}
		if len(pendingAudio) > 0 {
			if err := session.SendAudio(ctx, pendingAudio, true); err != nil {
				session.Close()
				session = nil
				pendingAudio = nil
				return err
			}
			pendingAudio = nil
		} else if err := session.SendAudio(ctx, nil, true); err != nil {
			session.Close()
			session = nil
			return err
		}
		err := <-resultsDone
		<-resultsForwarded
		session.Close()
		session = nil
		return err
	}

	// Process input stream
	for {

		chunk, err := input.Next()
		if err != nil {
			if !errors.Is(err, genx.ErrDone) && !errors.Is(err, io.EOF) {
				if session != nil {
					session.Close()
				}
				output.CloseWithError(err)
				return
			}
			// EOF: finish current session
			if err := finishSession(); err != nil {
				output.CloseWithError(err)
				return
			}
			return
		}

		if chunk == nil {
			continue
		}

		lastChunk = chunk

		// Check for EoS marker with audio MIME type
		if chunk.IsEndOfStream() {
			if blob, ok := chunk.Part.(*genx.Blob); ok && isAudioMIME(blob.MIMEType) {
				// Audio EoS: finish current session, emit text EoS
				if err := finishSession(); err != nil {
					output.CloseWithError(err)
					return
				}
				eosChunk := genx.NewTextEndOfStream()
				eosChunk.Role = lastChunk.Role
				eosChunk.Name = lastChunk.Name
				if err := output.Push(eosChunk); err != nil {
					return
				}
				continue
			}
			// Non-audio EoS: pass through
			if err := output.Push(chunk); err != nil {
				return
			}
			continue
		}

		// Handle audio blob
		if blob, ok := chunk.Part.(*genx.Blob); ok && isAudioMIME(blob.MIMEType) {
			// Start session on first audio chunk
			if session == nil {
				if err := startSession(); err != nil {
					output.CloseWithError(err)
					return
				}
			}
			if len(blob.Data) > 0 {
				for audio := range splitDoubaoASRAudio(blob.Data, t.audioChunkSize()) {
					if len(pendingAudio) > 0 {
						if err := session.SendAudio(ctx, pendingAudio, false); err != nil {
							session.Close()
							output.CloseWithError(err)
							return
						}
					}
					pendingAudio = audio
				}
			}
		} else {
			// Non-audio chunk: pass through
			if err := output.Push(chunk); err != nil {
				return
			}
		}
	}
}

func (t *DoubaoASRSAUC) openSession(ctx context.Context) (doubaoASRSession, error) {
	format := t.format
	if format == "ogg" {
		format = string(doubaospeech.FormatOGG)
	}

	config := &doubaospeech.ASRV2Config{
		Format:     doubaospeech.AudioFormat(format),
		SampleRate: doubaospeech.SampleRate(t.sampleRate),
		Channels:   t.channels,
		Bits:       t.bits,
		Language:   doubaospeech.Language(t.language),
		EnableITN:  t.enableITN,
		EnablePunc: t.enablePunc,
		Hotwords:   t.hotwords,
		ResultType: t.resultType,
		ResourceID: t.resourceID,
	}
	return t.client.ASRV2.OpenStreamSession(ctx, config)
}

func (t *DoubaoASRSAUC) audioChunkSize() int {
	if t.chunkSize > 0 {
		return t.chunkSize
	}
	if strings.EqualFold(t.format, "pcm") {
		bytesPerSample := t.bits / 8
		if bytesPerSample <= 0 {
			bytesPerSample = 2
		}
		channels := t.channels
		if channels <= 0 {
			channels = 1
		}
		sampleRate := t.sampleRate
		if sampleRate <= 0 {
			sampleRate = 16000
		}
		return sampleRate * bytesPerSample * channels / 10
	}
	return 256
}

func splitDoubaoASRAudio(data []byte, chunkSize int) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		if chunkSize <= 0 {
			chunkSize = 256
		}
		for offset := 0; offset < len(data); offset += chunkSize {
			end := min(offset+chunkSize, len(data))
			if !yield(data[offset:end]) {
				return
			}
		}
	}
}

func (t *DoubaoASRSAUC) receiveResults(session doubaoASRSession, lastChunk *genx.MessageChunk, resultsCh chan<- *genx.MessageChunk, done chan<- error) {
	defer close(resultsCh)

	// Track processed utterances by end time to avoid duplicates
	lastEndTime := 0
	resultCount := 0
	textCount := 0
	lastText := ""
	lastUtteranceCount := 0
	lastFinal := false

	for result, err := range session.Recv() {
		if err != nil {
			done <- err
			return
		}
		resultCount++
		lastText = result.Text
		lastUtteranceCount = len(result.Utterances)
		lastFinal = result.IsFinal

		// Process definite utterances from the utterances array
		emittedResultText := false
		if len(result.Utterances) > 0 {
			for _, utt := range result.Utterances {
				if utt.Definite && utt.EndTime > lastEndTime && utt.Text != "" {
					outChunk := &genx.MessageChunk{
						Part: genx.Text(utt.Text),
					}
					if lastChunk != nil {
						outChunk.Role = lastChunk.Role
						outChunk.Name = lastChunk.Name
					}
					resultsCh <- outChunk
					lastEndTime = utt.EndTime
					textCount++
					emittedResultText = true
				}
			}
		}
		if !emittedResultText && result.IsFinal && result.Text != "" {
			outChunk := &genx.MessageChunk{
				Part: genx.Text(result.Text),
			}
			if lastChunk != nil {
				outChunk.Role = lastChunk.Role
				outChunk.Name = lastChunk.Name
			}
			resultsCh <- outChunk
			textCount++
		}
	}
	if textCount == 0 {
		done <- fmt.Errorf("doubao asr returned no text: results=%d last_final=%t last_text=%q last_utterances=%d", resultCount, lastFinal, lastText, lastUtteranceCount)
		return
	}
	done <- nil
}

// isAudioMIME checks if a MIME type is audio
func isAudioMIME(mimeType string) bool {
	return len(mimeType) >= 6 && mimeType[:6] == "audio/"
}
