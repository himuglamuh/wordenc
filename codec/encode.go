package codec

import (
	"encoding/binary"
	"fmt"
)

const (
	bitsPerWord = 11
	wordMask    = (1 << bitsPerWord) - 1
)

// 8-byte magic: "WENC" + ver + flags + poison + '|'
var MagicV1 = [8]byte{'W', 'E', 'N', 'C', 0x01, 0x00, 0xA7, '|'}

type Encoder struct {
	Words []string
	// if Framed is true, encoder writes MAGIC + uvarint(len) + payload
	Framed bool
}

func NewEncoder(words []string, framed bool) (*Encoder, error) {
	if len(words) != 2048 {
		return nil, fmt.Errorf("wordlist must be 2048 words, got %d", len(words))
	}
	return &Encoder{Words: words, Framed: framed}, nil
}

func (e *Encoder) Encode(data []byte) []string {
	var blob []byte
	if e.Framed {
		var lenBuf [binary.MaxVarintLen64]byte
		n := binary.PutUvarint(lenBuf[:], uint64(len(data)))
		blob = make([]byte, 0, len(MagicV1)+n+len(data))
		blob = append(blob, MagicV1[:]...)
		blob = append(blob, lenBuf[:n]...)
		blob = append(blob, data...)
	} else {
		blob = data
	}
	indices := packToIndices(blob)
	out := make([]string, len(indices))
	for i, v := range indices {
		out[i] = e.Words[v]
	}
	return out
}

func packToIndices(blob []byte) []int {
	var (
		acc  uint64
		accN uint
		out  []int
	)
	for _, b := range blob {
		acc = (acc << 8) | uint64(b)
		accN += 8
		for accN >= bitsPerWord {
			accN -= bitsPerWord
			v := (acc >> accN) & wordMask
			out = append(out, int(v))
		}
	}
	if accN > 0 {
		// pad with zeros
		v := (acc << (bitsPerWord - accN)) & wordMask
		out = append(out, int(v))
	}
	return out
}
