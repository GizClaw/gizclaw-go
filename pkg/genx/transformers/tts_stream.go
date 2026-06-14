package transformers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/GizClaw/gizclaw-go/pkg/genx"
)

const defaultTTSSegmentMaxRunes = 256

type ttsChunkMeta struct {
	Role     genx.Role
	Name     string
	StreamID string
}

type ttsSynthesizer func(context.Context, string, ttsChunkMeta, string, *bufferStream) error

func runTTSTransform(ctx context.Context, input genx.Stream, output *bufferStream, mimeType string, synthesize ttsSynthesizer) {
	defer output.Close()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	segmenter := newTTSSentenceSegmenter(defaultTTSSegmentMaxRunes)
	var meta ttsChunkMeta

	flush := func(all bool) error {
		for _, segment := range segmenter.Segments(all) {
			if strings.TrimSpace(segment) == "" {
				continue
			}
			if err := synthesize(ctx, segment, meta, mimeType, output); err != nil {
				return err
			}
		}
		return nil
	}

	for {
		chunk, err := input.Next()
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, genx.ErrDone) {
				output.CloseWithError(err)
				return
			}
			if err := flush(true); err != nil {
				output.CloseWithError(err)
			}
			return
		}
		if chunk == nil {
			continue
		}

		updateTTSMeta(&meta, chunk)

		text, isText := chunk.Part.(genx.Text)
		if isText {
			if text != "" {
				segmenter.WriteString(string(text))
				if err := flush(false); err != nil {
					output.CloseWithError(err)
					return
				}
			}
			if chunk.IsEndOfStream() {
				if err := flush(true); err != nil {
					output.CloseWithError(err)
					return
				}
				if err := output.Push(newTTSEOSChunk(meta, mimeType)); err != nil {
					return
				}
				segmenter.Reset()
				meta = ttsChunkMeta{}
			}
			continue
		}

		if err := flush(true); err != nil {
			output.CloseWithError(err)
			return
		}
		if err := output.Push(chunk); err != nil {
			return
		}
	}
}

func updateTTSMeta(meta *ttsChunkMeta, chunk *genx.MessageChunk) {
	meta.Role = chunk.Role
	meta.Name = chunk.Name
	if chunk.Ctrl != nil && chunk.Ctrl.StreamID != "" {
		meta.StreamID = chunk.Ctrl.StreamID
	}
}

func pushTTSAudioChunk(output *bufferStream, meta ttsChunkMeta, mimeType string, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	chunk := &genx.MessageChunk{
		Role: meta.Role,
		Name: meta.Name,
		Part: &genx.Blob{
			MIMEType: mimeType,
			Data:     normalizeTTSAudio(mimeType, data),
		},
	}
	if meta.StreamID != "" {
		chunk.Ctrl = &genx.StreamCtrl{StreamID: meta.StreamID}
	}
	return output.Push(chunk)
}

func newTTSEOSChunk(meta ttsChunkMeta, mimeType string) *genx.MessageChunk {
	return &genx.MessageChunk{
		Role: meta.Role,
		Name: meta.Name,
		Part: &genx.Blob{MIMEType: mimeType},
		Ctrl: &genx.StreamCtrl{StreamID: meta.StreamID, EndOfStream: true},
	}
}

func normalizeTTSAudio(mimeType string, data []byte) []byte {
	switch strings.ToLower(mimeType) {
	case "audio/mpeg", "audio/mp3", "audio/x-mpeg":
		return stripID3Tags(data)
	default:
		return data
	}
}

// TTSAudioNormalizer removes provider container headers that are unsafe to
// concatenate across multiple synthesized text segments.
type TTSAudioNormalizer struct {
	mimeType string
	pending  []byte
}

// NewTTSAudioNormalizer creates a streaming audio normalizer for TTS output.
func NewTTSAudioNormalizer(mimeType string) *TTSAudioNormalizer {
	return &TTSAudioNormalizer{mimeType: strings.ToLower(mimeType)}
}

func newTTSAudioNormalizer(mimeType string) *TTSAudioNormalizer {
	return NewTTSAudioNormalizer(mimeType)
}

func (n *TTSAudioNormalizer) Write(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	if !n.isMP3() {
		return data
	}
	n.pending = append(n.pending, data...)
	return n.drain(false)
}

func (n *TTSAudioNormalizer) Flush() []byte {
	if !n.isMP3() {
		return nil
	}
	return n.drain(true)
}

func (n *TTSAudioNormalizer) isMP3() bool {
	switch n.mimeType {
	case "audio/mpeg", "audio/mp3", "audio/x-mpeg":
		return true
	default:
		return false
	}
}

func (n *TTSAudioNormalizer) drain(final bool) []byte {
	var out []byte
	for len(n.pending) > 0 {
		if bytes.HasPrefix(n.pending, []byte("ID3")) {
			size, valid, complete := id3v2TagSizeState(n.pending)
			if valid && !complete {
				if !final {
					return out
				}
				n.pending = nil
				return out
			}
			if !valid {
				if !final && len(n.pending) < 10 {
					return out
				}
				out = append(out, n.pending[0])
				n.pending = n.pending[1:]
				continue
			}
			n.pending = n.pending[size:]
			continue
		}

		idx := bytes.Index(n.pending, []byte("ID3"))
		if idx < 0 {
			keep := 2
			if final {
				out = append(out, n.pending...)
				n.pending = nil
				return out
			}
			if len(n.pending) <= keep {
				return out
			}
			emit := len(n.pending) - keep
			out = append(out, n.pending[:emit]...)
			n.pending = n.pending[emit:]
			return out
		}
		if idx > 0 {
			out = append(out, n.pending[:idx]...)
			n.pending = n.pending[idx:]
			continue
		}
	}
	return out
}

func stripID3Tags(data []byte) []byte {
	data = stripID3v1Tag(data)
	if !bytes.Contains(data, []byte("ID3")) {
		return data
	}
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		if size, ok := id3v2TagSize(data[i:]); ok {
			i += size
			continue
		}
		out = append(out, data[i])
		i++
	}
	return out
}

func stripID3v1Tag(data []byte) []byte {
	const id3v1Size = 128
	if len(data) >= id3v1Size && bytes.Equal(data[len(data)-id3v1Size:len(data)-id3v1Size+3], []byte("TAG")) {
		return data[:len(data)-id3v1Size]
	}
	return data
}

func id3v2TagSize(data []byte) (int, bool) {
	size, valid, complete := id3v2TagSizeState(data)
	return size, valid && complete
}

func id3v2TagSizeState(data []byte) (size int, valid bool, complete bool) {
	if len(data) < 10 || !bytes.Equal(data[:3], []byte("ID3")) {
		return 0, false, false
	}
	for _, b := range data[6:10] {
		if b&0x80 != 0 {
			return 0, false, false
		}
	}
	size = int(data[6])<<21 | int(data[7])<<14 | int(data[8])<<7 | int(data[9])
	total := 10 + size
	if data[5]&0x10 != 0 {
		total += 10
	}
	if total > len(data) {
		return total, true, false
	}
	return total, true, true
}

type ttsSentenceSegmenter struct {
	buf                string
	maxRunesPerSegment int
	firstSegment       bool
}

func newTTSSentenceSegmenter(maxRunes int) *ttsSentenceSegmenter {
	if maxRunes <= 0 {
		maxRunes = defaultTTSSegmentMaxRunes
	}
	return &ttsSentenceSegmenter{
		maxRunesPerSegment: maxRunes,
		firstSegment:       true,
	}
}

func (s *ttsSentenceSegmenter) WriteString(text string) {
	s.buf += text
}

func (s *ttsSentenceSegmenter) Reset() {
	s.buf = ""
	s.firstSegment = true
}

func (s *ttsSentenceSegmenter) Segments(all bool) []string {
	var segments []string
	for s.buf != "" {
		prefix, full := prefixRunes(s.buf, s.maxRunesPerSegment)
		idx := 0
		if s.firstSegment {
			idx = firstSentenceBoundaryIndex(prefix)
		} else {
			idx = lastSentenceBoundaryIndex(prefix)
		}
		switch {
		case idx > 0:
			segments = append(segments, s.buf[:idx])
			s.buf = s.buf[idx:]
			s.firstSegment = false
		case full:
			segments = append(segments, prefix)
			s.buf = s.buf[len(prefix):]
			s.firstSegment = false
		case all:
			segments = append(segments, s.buf)
			s.Reset()
		default:
			return segments
		}
	}
	return segments
}

func prefixRunes(text string, maxRunes int) (string, bool) {
	if maxRunes <= 0 {
		return text, false
	}
	count := 0
	for idx := range text {
		if count == maxRunes {
			return text[:idx], true
		}
		count++
	}
	return text, count >= maxRunes
}

func firstSentenceBoundaryIndex(text string) int {
	return sentenceBoundaryIndex(text, false)
}

func lastSentenceBoundaryIndex(text string) int {
	return sentenceBoundaryIndex(text, true)
}

func sentenceBoundaryIndex(text string, last bool) int {
	type runeInfo struct {
		value rune
		end   int
	}
	runes := make([]runeInfo, 0, utf8.RuneCountInString(text))
	for idx, r := range text {
		runes = append(runes, runeInfo{value: r, end: idx + utf8.RuneLen(r)})
	}
	found := 0
	for i, info := range runes {
		prev := rune(0)
		if i > 0 {
			prev = runes[i-1].value
		}
		next := rune(0)
		if i < len(runes)-1 {
			next = runes[i+1].value
		}
		if !isTTSSentenceBoundary(info.value, prev, next) {
			continue
		}
		if !last {
			return info.end
		}
		found = info.end
	}
	return found
}

func isTTSSentenceBoundary(r, prev, next rune) bool {
	switch r {
	case '.', ':', ',', '：':
		if unicode.IsNumber(prev) && unicode.IsNumber(next) {
			return false
		}
		return true
	case '，', '；', '。', '？', '！', '…', '～',
		'?', '!', '¿', '¡', ';', '~',
		'\r', '\n', '„', '・':
		return true
	default:
		return false
	}
}
