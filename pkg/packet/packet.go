package packet

import (
    "bytes"
    "encoding/binary"
    "io"
)

// Wrapper
type reader struct {
    io.Reader
}

func(r reader) ReadByte() (byte, error) {
    p := make([]byte, 1)
    _, err := r.Reader.Read(p)
    if err != nil {
        return 0, err
    }
    return p[0], nil
}

type Reader interface {
    io.Reader
    io.ByteReader
}

// PacketReader
type PacketReader struct {
    r Reader
}

func NewPacketReader(id int, ir io.Reader) *PacketReader {

    var r Reader = reader{ir}

    l := ReadVarInt(r)
    p := make([]byte, l)
    n, err := io.ReadFull(r, p)
    if err != nil {
        panic(err)
    }
    if l != int32(n) {
        panic("Wrong packet length")
    }

    r = bytes.NewReader(p)
    id1 := ReadVarInt(r)
    if id != int(id1) {
        panic("Wrong packet id")
    }

    return &PacketReader{r}

}

func(pr *PacketReader) readFull(p []byte) {
    n, err := io.ReadFull(pr.r, p)
    if err != nil {
        panic(err)
    }
    if n != len(p) {
        panic("Read length mismatch")
    }
}

func(pr *PacketReader) NextVarInt() int32 {
    return ReadVarInt(pr.r)
}

func(pr *PacketReader) NextInt(sz int) int64 {
    p := make([]byte, sz)
    pr.readFull(p)

    be := binary.BigEndian
    x := int64(0)
    switch sz {
    case 2: x = int64(be.Uint16(p))
    case 4: x = int64(be.Uint32(p))
    case 8: x = int64(be.Uint64(p))
    }
    return x
}

func(pr *PacketReader) NextString() string {
    l := pr.NextVarInt()
    p := make([]byte, l)
    pr.readFull(p)
    return string(p)
}

// Packet
type Packet struct {
    id int
    data []byte
}

func NewPacket(id int) *Packet {
    return &Packet{id, make([]byte, 0)}
}

func(pk *Packet) Bytes() []byte {
    whole := append(VarInt(int32(pk.id)), pk.data...)
    return append(VarInt(int32(len(whole))), whole...)
}

func(pk *Packet) put(p []byte) {
    pk.data = append(pk.data, p...)
}

func(pk *Packet) PutVarInt(x int32) {
    pk.put(VarInt(x))
}

func(pk *Packet) PutInt(x int64, sz int) {
    p := make([]byte, sz)
    be := binary.BigEndian
    switch sz {
    case 2: be.PutUint16(p, uint16(x))
    case 4: be.PutUint32(p, uint32(x))
    case 8: be.PutUint64(p, uint64(x))
    default: panic("Wrong int size")
    }
    pk.put(p)
}

func(pk *Packet) PutString(s string) {
    p := []byte(s)
    pk.PutVarInt(int32(len(p)))
    pk.put(p)
}


// Varint
func varint(x uint64, sz int) []byte {
    var l int
    switch sz {
    case 4: l = binary.MaxVarintLen32
    case 8: l = binary.MaxVarintLen64
    default: panic("Wrong int size")
    }

    p := make([]byte, l)
    n := binary.PutUvarint(p, x)
    return p[:n]
}

func VarInt(x int32) []byte {
    return varint(uint64(x) << 32 >> 32, 4)
}

func VarLong(x int64) []byte {
    return varint(uint64(x), 8)
}

func readVarint(r io.ByteReader) uint64 {
    n, err := binary.ReadUvarint(r)
    if err != nil {
        panic(err)
    }
    return n
}

func ReadVarInt(r io.ByteReader) int32 {
    return int32(readVarint(r))
}

func ReadVarLong(r io.ByteReader) int64 {
    return int64(readVarint(r))
}
