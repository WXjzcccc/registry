package registry

import (
	"encoding/binary"
	"io"
)

type namedKey struct {
	binOffset int64
	fpOffset  int64
	rws       io.ReadWriteSeeker

	signature string // must be equal to "nk"
	flags     uint16

	lastModified uint64

	parentKeyOffset         uint32 // The offset value is in bytes and relative from the start of the hive bin data
	numberOfSubKeys         uint32
	numberOfVolatileSubKeys uint32

	subKeysListOffset  uint32 // The offset value is in bytes and relative from the start of the hive bin data. Refers to a sub keys list or contains -1 (0xffffffff) if empty.
	volatileListOffset uint32 // The offset value is in bytes and relative from the start of the hive bin data.

	numberOfValues   uint32
	valuesListOffset uint32 // The offset value is in bytes and relative from the start of the hive bin data.

	securityKeyOffset uint32 // The offset value is in bytes and relative from the start of the hive bin data.

	classNameOffset uint32 // The offset value is in bytes and relative from the start of the hive bin data.

	largestSubKeyNameSize      uint32
	largestSubKeyClassNameSize uint32
	largestValueNameSize       uint32
	largestValueDataSize       uint32

	keyNameSize   uint16
	classNameSize uint16

	name       string
	className  string
	headerSize int64

	values *valueList
}

func newNamedKey(rws io.ReadWriteSeeker, binOffset int64, fpOffset int64) *namedKey {
	return &namedKey{
		rws:       rws,
		binOffset: binOffset,
		fpOffset:  fpOffset,
	}
}

func (nk *namedKey) validate() error {
	if nk.signature != namedKeySig {
		return errorW{err: ErrCorruptRegistry, cause: errBadSignature, function: "namedKey.validate()"}
	}

	return nil
}

func (nk *namedKey) Read() error {
	r := nk.rws
	_, err := r.Seek(nk.fpOffset, io.SeekStart)
	if err != nil {
		return errorW{err: ErrCorruptRegistry, cause: err, function: "namedKey.Read() r.Seek"}
	}

	buf := make([]byte, 76)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return errorW{err: ErrCorruptRegistry, cause: err, function: "namedKey.Read() io.ReadFull"}
	}

	nk.signature = string(buf[0:2])
	nk.flags = binary.LittleEndian.Uint16(buf[2:4])
	nk.lastModified = binary.LittleEndian.Uint64(buf[4:12])
	nk.parentKeyOffset = binary.LittleEndian.Uint32(buf[16:20])
	nk.numberOfSubKeys = binary.LittleEndian.Uint32(buf[20:24])
	nk.numberOfVolatileSubKeys = binary.LittleEndian.Uint32(buf[24:28])
	nk.subKeysListOffset = binary.LittleEndian.Uint32(buf[28:32])
	nk.volatileListOffset = binary.LittleEndian.Uint32(buf[32:36])
	nk.numberOfValues = binary.LittleEndian.Uint32(buf[36:40])
	nk.valuesListOffset = binary.LittleEndian.Uint32(buf[40:44])
	nk.securityKeyOffset = binary.LittleEndian.Uint32(buf[44:48])
	nk.classNameOffset = binary.LittleEndian.Uint32(buf[48:52])
	nk.largestSubKeyNameSize = binary.LittleEndian.Uint32(buf[52:56])
	nk.largestSubKeyClassNameSize = binary.LittleEndian.Uint32(buf[56:60])
	nk.largestValueNameSize = binary.LittleEndian.Uint32(buf[60:64])
	nk.largestValueDataSize = binary.LittleEndian.Uint32(buf[64:68])
	nk.keyNameSize = binary.LittleEndian.Uint16(buf[72:74])
	nk.classNameSize = binary.LittleEndian.Uint16(buf[74:76])

	buf = make([]byte, nk.keyNameSize)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return errorW{err: ErrCorruptRegistry, cause: err, function: "namedKey.Read() io.ReadFull"}
	}
	//nk.name = string(buf)
	myName, err := utf16leBytesToString(buf)
	if err != nil {
		return err
	}
	nk.name = myName
	loc, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return errorW{err: ErrCorruptRegistry, cause: err, function: "namedKey.Read() r.Seek"}
	}

	// seek to end of padding
	for loc%8 != 0 {
		loc, err = r.Seek(1, io.SeekCurrent)
		if err != nil {
			return errorW{err: ErrCorruptRegistry, cause: err, function: "namedKey.Read() r.Seek"}
		}
	}

	nk.headerSize = int64(4096)

	nk.values = newValueList(r, nk.binOffset, nk.valuesListOffset, nk.numberOfValues)
	err = nk.values.Read()
	if err != nil {
		return err
	}

	_, err = r.Seek(nk.binOffset+int64(nk.classNameOffset), io.SeekStart)
	if err != nil {
		return err
	}
	classNameBytes := make([]byte, nk.classNameSize)
	_, err = io.ReadFull(r, classNameBytes)
	if err != nil {
		return err
	}
	nk.className, err = utf16leBytesToString(classNameBytes)
	if err != nil {
		return err
	}
	return nk.validate()
}
