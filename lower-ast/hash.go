package lower_ast

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
)

type Hashable interface {
	Hash() uint64
}

type hashBuilder struct {
	h hash.Hash64
}

func newHashBuilder() hashBuilder {
	return hashBuilder{h: fnv.New64a()}
}

func newHash(typ string) hashBuilder {
	h := hashBuilder{h: fnv.New64a()}
	h.String(typ)
	return h
}

func (b hashBuilder) Literial(v any) hashBuilder {
	switch v := v.(type) {
	default:
		panic(fmt.Errorf("unknown type: %T:%v", v, v))
	case nil:
		b.Nil()
	case byte:
		b.Byte(v)
	case uint64:
		b.Uint64(v)
	case string:
		b.String(v)
	case bool, int, int32, int64, uint, uint32, float32, float64:
		_, _ = b.h.Write([]byte(fmt.Sprintf("%T:%v", v, v)))
	}
	return b
}

func (b hashBuilder) Nil() hashBuilder {
	b.h.Write([]byte{0})
	return b
}

func (b hashBuilder) Byte(v byte) hashBuilder {
	var buf [1]byte
	buf[0] = v
	b.h.Write([]byte{1})
	_, _ = b.h.Write(buf[:])
	return b
}

func (b hashBuilder) Uint64(v uint64) hashBuilder {
	var buf [8]byte
	b.h.Write([]byte{2})
	binary.LittleEndian.PutUint64(buf[:], v)
	_, _ = b.h.Write(buf[:])
	return b
}

func (b hashBuilder) String(v string) hashBuilder {
	b.h.Write([]byte{3})
	_, _ = b.h.Write([]byte(v))
	return b
}

func (b hashBuilder) Sum64() uint64 {
	return b.h.Sum64()
}
