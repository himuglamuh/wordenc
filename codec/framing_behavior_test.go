package codec

import (
	"bytes"
	"testing"
)

func TestFramed_Truncated_ReturnsPartial(t *testing.T) {
	words, index := mustCodec(t)

	enc, _ := NewEncoder(words, true)
	dec := NewDecoder(index)
	dec.AllowPartial = true

	in := []byte("hello world this is a longer message")
	w := enc.Encode(in)

	// simulate user missing the last few words
	if len(w) < 6 {
		t.Fatalf("unexpectedly short encoding")
	}
	wTrunc := w[:len(w)-3]

	got, err := dec.Decode(wTrunc, DecodeAuto)
	if err == nil {
		// It *might* sometimes land exactly on boundaries and not error,
		// but generally truncation should surface ErrTruncated.
		// We won't require the error, but we require "got is a prefix".
	} else if err != ErrTruncated {
		t.Fatalf("expected ErrTruncated (or nil), got %v", err)
	}

	if len(got) == 0 {
		t.Fatalf("expected partial output, got empty")
	}
	if !bytes.HasPrefix(in, got) {
		t.Fatalf("expected output to be prefix of input; got=%q wantprefixof=%q", string(got), string(in))
	}
}

func TestFramed_ExtraWords_Ignored(t *testing.T) {
	words, index := mustCodec(t)

	encFramed, _ := NewEncoder(words, true)
	encRaw, _ := NewEncoder(words, false)
	dec := NewDecoder(index)

	in := []byte("hello world")
	w := encFramed.Encode(in)

	// append some extra raw words (junk after payload)
	junk := encRaw.Encode([]byte("junkjunkjunk"))
	w2 := append(append([]string{}, w...), junk...)

	got, derr := dec.Decode(w2, DecodeAuto)
	if derr != nil {
		t.Fatalf("Decode(auto) error: %v", derr)
	}
	if !bytes.Equal(got, in) {
		t.Fatalf("mismatch: got %q want %q", string(got), string(in))
	}
}

func TestFramed_MaxLen_GuardsAbsurd(t *testing.T) {
	words, index := mustCodec(t)

	enc, _ := NewEncoder(words, true)
	dec := NewDecoder(index)

	in := make([]byte, 1024) // 1KB
	w := enc.Encode(in)

	dec.MaxLen = 16 // absurdly small: will reject framed and auto should fall back to RAW

	got, derr := dec.Decode(w, DecodeAuto)
	if derr != nil {
		// auto returns raw on ErrAbsurdLength in our decoder; should not error
		t.Fatalf("Decode(auto) unexpected error: %v", derr)
	}
	// In fallback-to-RAW case, output will include header+varint+payload bytes,
	// so it should NOT equal the original payload.
	if bytes.Equal(got, in) {
		t.Fatalf("expected not equal due to framed rejection + raw fallback")
	}
	if len(got) <= len(in) {
		t.Fatalf("expected raw fallback to be larger than payload; got=%d payload=%d", len(got), len(in))
	}
}
