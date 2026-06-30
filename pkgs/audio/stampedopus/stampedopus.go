package stampedopus

import "encoding/binary"

const (
	// Version 是 stampedopus wire format 的版本号。
	Version byte = 1

	// HeaderSize 是 stampedopus 头部固定长度：1 字节版本 + 7 字节时间戳。
	HeaderSize = 8

	timestampBytes = HeaderSize - 1
)

const timestampMask uint64 = (1 << (timestampBytes * 8)) - 1

// Pack 把毫秒时间戳和 opus frame 打包成 stamped frame。
//
// 编码格式：
//   - byte[0]   : Version
//   - byte[1:8] : Timestamp（uint64 低 56 bit，Big-endian）
//   - byte[8:]  : 原始 opus frame
//
// API 合同：
//   - Pack 允许 frame 为 nil 或空切片，此时会产出仅含 8 字节 header 的报文。
//   - 该报文会被 Unpack 判定为无效（ok=false），因为 payload 为空。
//   - 若调用方期望后续 Unpack 成功，必须保证 frame 非空。
func Pack(timestamp uint64, frame []byte) []byte {
	out := make([]byte, HeaderSize+len(frame))
	out[0] = Version

	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], timestamp&timestampMask)
	copy(out[1:HeaderSize], ts[1:])

	copy(out[HeaderSize:], frame)
	return out
}

// Unpack 解包 stamped frame。
//
// 返回值：
//   - timestamp: 解码出的毫秒时间戳（仅低 56 bit）
//   - frame:     opus frame 拷贝
//   - ok:        是否成功解包
func Unpack(data []byte) (timestamp uint64, frame []byte, ok bool) {
	if len(data) < HeaderSize {
		return 0, nil, false
	}
	if data[0] != Version {
		return 0, nil, false
	}

	payload := data[HeaderSize:]
	if len(payload) == 0 {
		return 0, nil, false
	}

	var ts [8]byte
	copy(ts[1:], data[1:HeaderSize])
	timestamp = binary.BigEndian.Uint64(ts[:])

	frame = append([]byte(nil), payload...)
	return timestamp, frame, true
}
