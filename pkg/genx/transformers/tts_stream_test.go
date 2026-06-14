package transformers

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/genx"
)

func TestTTSSentenceSegmenterSplitsOnSemanticBoundaries(t *testing.T) {
	segmenter := newTTSSentenceSegmenter(256)
	segmenter.WriteString("你好，我的朋友。3.14 是一个小数，10:15 是时间")

	got := segmenter.Segments(false)
	want := []string{"你好，", "我的朋友。3.14 是一个小数，"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Segments(false) = %#v, want %#v", got, want)
	}

	got = segmenter.Segments(true)
	want = []string{"10:15 是时间"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Segments(true) = %#v, want %#v", got, want)
	}
}

func TestTTSSentenceSegmenterSplitsAtMaxRunes(t *testing.T) {
	segmenter := newTTSSentenceSegmenter(3)
	segmenter.WriteString("一二三四五")

	got := segmenter.Segments(false)
	want := []string{"一二三"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Segments(false) = %#v, want %#v", got, want)
	}

	got = segmenter.Segments(true)
	want = []string{"四五"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Segments(true) = %#v, want %#v", got, want)
	}
}

func TestNormalizeTTSAudioStripsRepeatedID3Tags(t *testing.T) {
	data := append(fakeID3Header(), []byte("frame-a")...)
	data = append(data, fakeID3Header()...)
	data = append(data, []byte("frame-b")...)

	got := normalizeTTSAudio("audio/mpeg", data)
	if bytes.Contains(got, []byte("ID3")) {
		t.Fatalf("expected ID3 tags to be stripped from %q", got)
	}
	if string(got) != "frame-aframe-b" {
		t.Fatalf("normalizeTTSAudio() = %q, want frame-aframe-b", got)
	}
}

func TestTTSAudioNormalizerStripsID3AcrossChunkBoundaries(t *testing.T) {
	normalizer := newTTSAudioNormalizer("audio/mpeg")
	var got []byte
	first := append(fakeID3Tag([]byte("tag-a")), []byte("frame-a")...)
	second := append(fakeID3Tag([]byte("tag-b")), []byte("frame-b")...)
	for _, chunk := range [][]byte{
		first[:2],
		first[2:10],
		first[10:13],
		append(first[13:], second[:2]...),
		second[2:],
	} {
		got = append(got, normalizer.Write(chunk)...)
	}
	got = append(got, normalizer.Flush()...)
	if bytes.Contains(got, []byte("ID3")) {
		t.Fatalf("expected split ID3 tags to be stripped from %q", got)
	}
	if string(got) != "frame-aframe-b" {
		t.Fatalf("normalizer output = %q, want frame-aframe-b", got)
	}
}

func TestRunTTSTransformPreservesInputStreamIDAndNormalizesAudio(t *testing.T) {
	input := &testStream{chunks: []*genx.MessageChunk{
		{
			Role: genx.RoleUser,
			Name: "gear",
			Part: genx.Text("你好，世界。"),
			Ctrl: &genx.StreamCtrl{StreamID: "input-stream"},
		},
		{
			Role: genx.RoleUser,
			Name: "gear",
			Part: genx.Text(""),
			Ctrl: &genx.StreamCtrl{StreamID: "input-stream", EndOfStream: true},
		},
	}, doneErr: io.EOF}

	output := newBufferStream(8)
	var texts []string
	runTTSTransform(context.Background(), input, output, "audio/mpeg", func(_ context.Context, text string, meta ttsChunkMeta, mimeType string, out *bufferStream) error {
		texts = append(texts, text)
		data := append(fakeID3Header(), []byte("audio:"+text)...)
		return pushTTSAudioChunk(out, meta, mimeType, data)
	})

	if want := []string{"你好，", "世界。"}; !reflect.DeepEqual(texts, want) {
		t.Fatalf("synthesized texts = %#v, want %#v", texts, want)
	}

	chunks := collectTransformerChunks(t, output)
	if len(chunks) != 3 {
		t.Fatalf("got %d output chunks, want 3", len(chunks))
	}
	for i, chunk := range chunks[:2] {
		if chunk.Ctrl == nil || chunk.Ctrl.StreamID != "input-stream" {
			t.Fatalf("audio chunk %d StreamID = %#v, want input-stream", i, chunk.Ctrl)
		}
		blob, ok := chunk.Part.(*genx.Blob)
		if !ok {
			t.Fatalf("audio chunk %d part = %T, want *genx.Blob", i, chunk.Part)
		}
		if bytes.Contains(blob.Data, []byte("ID3")) {
			t.Fatalf("audio chunk %d still contains ID3 tag", i)
		}
		if chunk.Role != genx.RoleUser || chunk.Name != "gear" {
			t.Fatalf("audio chunk %d meta = role %q name %q", i, chunk.Role, chunk.Name)
		}
	}
	eos := chunks[2]
	if eos.Ctrl == nil || !eos.Ctrl.EndOfStream || eos.Ctrl.StreamID != "input-stream" {
		t.Fatalf("eos ctrl = %#v, want input-stream eos", eos.Ctrl)
	}
}

func TestRunTTSTransformDoesNotCreateStreamID(t *testing.T) {
	input := &testStream{chunks: []*genx.MessageChunk{
		{Part: genx.Text("hello.")},
		genx.NewTextEndOfStream(),
	}, doneErr: io.EOF}

	output := newBufferStream(4)
	runTTSTransform(context.Background(), input, output, "audio/mpeg", func(_ context.Context, _ string, meta ttsChunkMeta, mimeType string, out *bufferStream) error {
		return pushTTSAudioChunk(out, meta, mimeType, []byte("audio"))
	})

	chunks := collectTransformerChunks(t, output)
	if len(chunks) != 2 {
		t.Fatalf("got %d output chunks, want 2", len(chunks))
	}
	if chunks[0].Ctrl != nil {
		t.Fatalf("audio chunk ctrl = %#v, want nil", chunks[0].Ctrl)
	}
	if chunks[1].Ctrl == nil || !chunks[1].Ctrl.EndOfStream || chunks[1].Ctrl.StreamID != "" {
		t.Fatalf("eos ctrl = %#v, want eos without stream id", chunks[1].Ctrl)
	}
}

func collectTransformerChunks(t *testing.T, stream genx.Stream) []*genx.MessageChunk {
	t.Helper()
	var chunks []*genx.MessageChunk
	for {
		chunk, err := stream.Next()
		if err != nil {
			if err == io.EOF || err == genx.ErrDone {
				return chunks
			}
			t.Fatalf("Next() error = %v", err)
		}
		chunks = append(chunks, chunk)
	}
}

func fakeID3Header() []byte {
	return []byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0, 0}
}

func fakeID3Tag(payload []byte) []byte {
	header := fakeID3Header()
	size := len(payload)
	header[6] = byte((size >> 21) & 0x7f)
	header[7] = byte((size >> 14) & 0x7f)
	header[8] = byte((size >> 7) & 0x7f)
	header[9] = byte(size & 0x7f)
	return append(header, payload...)
}
