package codecconv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unsafe"

	"github.com/GizClaw/gizclaw-go/pkg/audio/codec/ogg"
	"github.com/GizClaw/gizclaw-go/pkg/audio/codec/opus"
)

// OggToPCM decodes an OGG/Opus stream into PCM at the requested Opus sample rate.
func OggToPCM(w io.Writer, r io.Reader, outputRate opus.OpusSampleRate) (int64, error) {
	if err := outputRate.Validate(); err != nil {
		return 0, err
	}

	decoderSampleRate := outputRate.Int()
	decoderChannels := 1

	var (
		dec           *opus.Decoder
		audioSeen     bool
		packetSeen    bool
		directWritten int64
	)
	defer func() {
		if dec != nil {
			_ = dec.Close()
		}
	}()

	for packet, err := range ogg.Packets(r) {
		if err != nil {
			return 0, fmt.Errorf("read ogg packets: %w", err)
		}
		packetSeen = true

		if IsOpusHeadPacket(packet.Data) {
			_, channels, err := ParseOpusHeadPacket(packet.Data)
			if err != nil {
				return 0, fmt.Errorf("parse opus head packet: %w", err)
			}
			decoderChannels = channels
			continue
		}
		if IsOpusTagsPacket(packet.Data) || len(packet.Data) == 0 {
			continue
		}

		if dec == nil {
			dec, err = opus.NewDecoder(decoderSampleRate, decoderChannels)
			if err != nil {
				return 0, fmt.Errorf("create opus decoder: %w", err)
			}
		}

		audioSeen = true
		maxFrameSize := (decoderSampleRate * 3) / 50
		if maxFrameSize <= 0 {
			return 0, fmt.Errorf("invalid max frame size from sample rate %d", decoderSampleRate)
		}

		samples, err := dec.Decode(packet.Data, maxFrameSize, false)
		if err != nil {
			return 0, fmt.Errorf("decode opus packet: %w", err)
		}
		n, err := io.Copy(w, bytes.NewReader(int16ToBytes(samples)))
		directWritten += n
		if err != nil {
			return 0, fmt.Errorf("write decoded pcm buffer: %w", err)
		}
	}

	if !packetSeen {
		return 0, errors.New("empty ogg packet stream")
	}
	if !audioSeen {
		return 0, errors.New("no opus audio packets found in ogg stream")
	}
	return directWritten, nil
}

// IsOpusHeadPacket reports whether packet is an OpusHead packet.
func IsOpusHeadPacket(packet []byte) bool {
	return len(packet) >= 8 && bytes.Equal(packet[:8], []byte("OpusHead"))
}

// IsOpusTagsPacket reports whether packet is an OpusTags packet.
func IsOpusTagsPacket(packet []byte) bool {
	return len(packet) >= 8 && bytes.Equal(packet[:8], []byte("OpusTags"))
}

// ParseOpusHeadPacket extracts sample rate and channels from an OpusHead packet.
func ParseOpusHeadPacket(packet []byte) (sampleRate, channels int, err error) {
	if !IsOpusHeadPacket(packet) {
		return 0, 0, errors.New("not an opus head packet")
	}
	if len(packet) < 19 {
		return 0, 0, fmt.Errorf("opus head packet too short: %d", len(packet))
	}

	channels = int(packet[9])
	if channels != 1 && channels != 2 {
		return 0, 0, fmt.Errorf("unsupported opus channels %d", channels)
	}

	sampleRate = int(binary.LittleEndian.Uint32(packet[12:16]))
	if sampleRate <= 0 {
		return 0, 0, fmt.Errorf("invalid opus sample rate %d", sampleRate)
	}

	return sampleRate, channels, nil
}

func int16ToBytes(data []int16) []byte {
	if len(data) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(data))), len(data)*2)
}
