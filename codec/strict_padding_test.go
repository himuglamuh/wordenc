package codec

import (
	"testing"
)

func TestStrictPadding_ErrsOnNonZeroPadBits(t *testing.T) {
	words, index := mustCodec(t)

	encRaw, _ := NewEncoder(words, false)
	dec := NewDecoder(index)
	dec.StrictPad = true

	// Create a small payload that will require padding bits.
	// RAW packing pads with zeros, which is fine. We need to *force* non-zero pad bits.
	// We can do that by taking a valid encoding and flipping the last word to another value.
	in := []byte{0xAA} // 10101010 (will not align cleanly to 11 bits)
	w := encRaw.Encode(in)
	if len(w) < 1 {
		t.Fatalf("expected at least 1 word")
	}

	// mutate last word to a different valid word, likely making leftover bits non-zero
	last := w[len(w)-1]
	var replacement string
	for cand := range index {
		if cand != last {
			replacement = cand
			break
		}
	}
	w[len(w)-1] = replacement

	_, err := dec.Decode(w, DecodeRaw)
	if err == nil {
		t.Fatalf("expected padding error, got nil")
	}
	if err != ErrBadPadding {
		t.Fatalf("expected ErrBadPadding, got %v", err)
	}
}
