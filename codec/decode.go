package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

type DecodeMode int

const (
	DecodeAuto DecodeMode = iota
	DecodeRaw
	DecodeFramed
)

type Decoder struct {
	Index        map[string]int
	MaxLen       uint64 // max declared payload length in framed mode (auto/framed)
	AllowPartial bool   // if true, return partial payload on truncation
	StrictPad    bool   // if true, error if leftover padding bits are non-zero
}

func NewDecoder(index map[string]int) *Decoder {
	return &Decoder{
		Index:        index,
		MaxLen:       256 << 20, // 256MB default safety
		AllowPartial: true,
		StrictPad:    false,
	}
}

// decodes a space/newline separated string of words, handles leading/trailing whitespace
func (d *Decoder) DecodeWords(input string, mode DecodeMode) ([]byte, error) {
	fields := strings.Fields(strings.TrimSpace(input))
	return d.Decode(fields, mode)
}

func (d *Decoder) Decode(words []string, mode DecodeMode) ([]byte, error) {
	indices := make([]int, 0, len(words))
	for _, w := range words {
		v, ok := d.Index[w]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrBadWord, w)
		}
		indices = append(indices, v)
	}

	rawBytes, padOK := unpackFromIndices(indices)
	if d.StrictPad && !padOK {
		return nil, ErrBadPadding
	}

	switch mode {
	case DecodeRaw:
		return rawBytes, nil
	case DecodeFramed:
		return d.decodeFramed(rawBytes)
	case DecodeAuto:
		if hasMagic(rawBytes) {
			// try framed; on implausible header fall back to raw
			b, err := d.decodeFramed(rawBytes)
			if err == nil {
				return b, nil
			}
			// only fall back on header/length plausibility issues
			if err == ErrBadHeader || err == ErrAbsurdLength {
				return rawBytes, nil
			}
			// truncation: if AllowPartial, decodeFramed already returned partial
			// otherwise propagate
			if err == ErrTruncated && d.AllowPartial {
				return b, nil
			}
			// for other errors, return raw as a forgiving default
			return rawBytes, nil
		}
		return rawBytes, nil
	default:
		return nil, fmt.Errorf("unknown decode mode")
	}
}

func hasMagic(b []byte) bool {
	return len(b) >= len(MagicV1) && bytes.Equal(b[:8], MagicV1[:])
}

func (d *Decoder) decodeFramed(b []byte) ([]byte, error) {
	if !hasMagic(b) {
		return nil, ErrBadHeader
	}
	r := bytes.NewReader(b[8:])

	L, err := binary.ReadUvarint(r)
	if err != nil {
		// if input is truncated, treat as header failure (auto can fall back)
		return nil, ErrBadHeader
	}
	if d.MaxLen > 0 && L > d.MaxLen {
		return nil, ErrAbsurdLength
	}

	remain := uint64(r.Len())
	if L > remain {
		if !d.AllowPartial {
			return nil, ErrTruncated
		}
		// return whatever is available (partial payload)
		payload := make([]byte, remain)
		_, _ = r.Read(payload)
		return payload, ErrTruncated
	}

	payload := make([]byte, L)
	_, _ = r.Read(payload)
	// if there are extra bytes after payload, ignore them in library
	// CLI can warn if it cares
	return payload, nil
}

func unpackFromIndices(indices []int) ([]byte, bool) {
	var (
		acc  uint64
		accN uint
		out  []byte
	)
	for _, v := range indices {
		acc = (acc << bitsPerWord) | uint64(v)
		accN += bitsPerWord
		for accN >= 8 {
			accN -= 8
			out = append(out, byte(acc>>accN))
			acc &= (1<<accN - 1)
		}
	}
	// leftover bits must be zero for clean padding
	if accN > 0 && (acc&(1<<accN-1)) != 0 {
		return out, false
	}
	return out, true
}
