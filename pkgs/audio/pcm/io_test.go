package pcm

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestWriteFuncAndDiscard(t *testing.T) {
	called := false
	w := WriteFunc(func(c Chunk) error {
		called = true
		if c.Format() != L16Mono16K {
			t.Fatalf("unexpected format: %v", c.Format())
		}
		return nil
	})

	if err := w.Write(L16Mono16K.DataChunk([]byte{1, 2})); err != nil {
		t.Fatalf("WriteFunc.Write() error: %v", err)
	}
	if !called {
		t.Fatal("WriteFunc.Write() should call underlying function")
	}

	if err := Discard.Write(L16Mono16K.DataChunk([]byte{3, 4})); err != nil {
		t.Fatalf("Discard.Write() error: %v", err)
	}
}

func TestIOWriterAndChunkWriter(t *testing.T) {
	var got []byte
	collector := WriteFunc(func(c Chunk) error {
		buf := bytes.Buffer{}
		_, err := c.WriteTo(&buf)
		if err != nil {
			return err
		}
		got = append(got, buf.Bytes()...)
		return nil
	})

	ioW := IOWriter(collector, L16Mono16K)
	written, err := ioW.Write([]byte{1, 2, 3, 4})
	if err != nil {
		t.Fatalf("IOWriter.Write() error: %v", err)
	}
	if written != 4 {
		t.Fatalf("IOWriter.Write() = %d, want 4", written)
	}
	if !bytes.Equal(got, []byte{1, 2, 3, 4}) {
		t.Fatalf("IOWriter wrote %v", got)
	}

	var out bytes.Buffer
	cw := ChunkWriter(&out)
	if err := cw.Write(L16Mono16K.DataChunk([]byte{9, 8, 7, 6})); err != nil {
		t.Fatalf("ChunkWriter.Write() error: %v", err)
	}
	if !bytes.Equal(out.Bytes(), []byte{9, 8, 7, 6}) {
		t.Fatalf("ChunkWriter output mismatch: %v", out.Bytes())
	}
}

func TestCopy(t *testing.T) {
	format := L16Mono16K
	minChunk := int(format.BytesInDuration(20 * 1e6)) // 20ms
	input := bytes.Repeat([]byte{1, 2}, minChunk/2+123)

	var copied []byte
	w := WriteFunc(func(c Chunk) error {
		buf := bytes.Buffer{}
		_, err := c.WriteTo(&buf)
		if err != nil {
			return err
		}
		copied = append(copied, buf.Bytes()...)
		return nil
	})

	if err := Copy(w, bytes.NewReader(input), format); err != nil {
		t.Fatalf("Copy() error: %v", err)
	}
	if !bytes.Equal(copied, input) {
		t.Fatalf("copied bytes mismatch")
	}
}

func TestCopyErrorPaths(t *testing.T) {
	format := L16Mono16K
	errWriter := errors.New("writer failed")

	w := WriteFunc(func(Chunk) error {
		return errWriter
	})
	minChunk := int(format.BytesInDuration(20 * 1e6))
	input := bytes.Repeat([]byte{1, 2}, minChunk/2)

	err := Copy(w, bytes.NewReader(input), format)
	if !errors.Is(err, errWriter) {
		t.Fatalf("Copy() writer err = %v, want %v", err, errWriter)
	}

	errReader := &failingReader{err: io.ErrNoProgress}
	err = Copy(Discard, errReader, format)
	if !errors.Is(err, io.ErrNoProgress) {
		t.Fatalf("Copy() reader err = %v, want %v", err, io.ErrNoProgress)
	}
}

type failingReader struct {
	err error
}

func (r *failingReader) Read([]byte) (int, error) {
	return 0, r.err
}
