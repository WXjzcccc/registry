package registry

import (
	"encoding/binary"
	"fmt"
	"time"
	"unicode"
	"unicode/utf16"
)

func date(i uint64) time.Time {
	return time.Unix(0, int64(i)*int64(time.Nanosecond))
}

func stringFromBytes(u []byte) string {
	b := make([]uint16, len(u)/2)
	for i := 0; i < len(u); i += 2 {
		b[i/2] = (uint16(u[i+1]) << 8) + uint16(u[i])
	}
	if b[len(u)/2-1] == 0 {
		b = b[:len(u)/2-1]
	}
	return string(utf16.Decode(b))
}

func stringsFromBytes(u []byte) (r []string) {
	str := make([]uint16, 0)
	for i := 0; i < len(u); i += 2 {
		c := (uint16(u[i+1]) << 8) + uint16(u[i]) // utf16-LE to rune
		if c == 0 && len(str) > 0 {               // end of string
			r = append(r, string(utf16.Decode(str)))
			str = make([]uint16, 0)
		} else { // append to cur string
			str = append(str, c)
		}
	}
	return
}

func uint64FromBytesLE(u []byte) uint64 {
	b := make([]byte, 8) // make sure b is uint64
	copy(b, u)
	return binary.LittleEndian.Uint64(b)
}

func uint32FromBytesBE(u []byte) uint32 {
	b := make([]byte, 4) // make sure b is uint32
	for i, v := range u {
		b[4-len(u)+i] = v
	}
	return binary.BigEndian.Uint32(b)
}

func bytesFromUint64LE(u uint64, dataType uint32) ([]byte, error) {
	switch dataType {
	case REG_DWORD:
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(u))
		return b, nil
	case REG_QWORD:
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, u)
		return b, nil
	default:
		return nil, fmt.Errorf("Invalid data type %v", Type(dataType))
	}
}

// bytesFromUint32BE returns the []byte representation of uint32
// ONLY USE FOR REG_DWORD_BIG_ENDIAN
func bytesFromUint32BE(u uint32) ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, u)
	return b, nil
}

func bytesFromStrings(strs []string) []byte {
	r := make([]byte, 0)
	for _, s := range strs {
		r = append(r, []byte(s)...)
		r = append(r, 0)
	}
	return r
}

func dataSizeFromType(u uint32) int {
	switch u {
	case REG_DWORD_LITTLE_ENDIAN, REG_DWORD_BIG_ENDIAN:
		return 4 // 32 bit
	case REG_QWORD:
		return 8 // 64 bit
	default:
		return 0
	}
}

func lhSubKeyHash(str string) uint32 {
	//var hashValue uint32 = 0
	//for idx := 0; idx < len(str); idx++ {
	//	hashValue *= 37
	//	hashValue += uint32(unicode.ToUpper(rune(str[idx])))
	//}
	// 修复出现乱码、中文的哈希计算情况，此算法能满足绝大多数情况，但仍有小部分情况出现哈希值与读取的不同，todo
	var hashValue uint32 = 0
	strRune := []rune(str)
	for idx := 0; idx < len(strRune); idx++ {
		hashValue *= 37
		hashValue += uint32(unicode.ToUpper(strRune[idx]))
	}
	return hashValue
}

func utf16leBytesToString(leBytes []byte) (string, error) {
	// 把字节数组转成字符串，注册表中的key在非ascii码时采用utf16-le进行编码
	var flag bool = false
	for _, b := range leBytes {
		if b > 0x7F || b < 0x20 {
			flag = true
		}
	}
	if !flag {
		return string(leBytes), nil
	}

	if len(leBytes)%2 != 0 {
		return "", fmt.Errorf("字节长度必须是 2 的倍数")
	}

	utf16Data := make([]uint16, len(leBytes)/2)
	for i := 0; i < len(utf16Data); i++ {
		utf16Data[i] = binary.LittleEndian.Uint16(leBytes[i*2:])
	}

	return string(utf16.Decode(utf16Data)), nil
}
