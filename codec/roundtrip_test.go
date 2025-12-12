package codec

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
)

func mustCodec(t *testing.T) ([]string, map[string]int) {
	t.Helper()
	words, index, err := LoadBIP39English()
	if err != nil {
		t.Fatalf("LoadBIP39English: %v", err)
	}
	return words, index
}

func TestRoundTrip_Framed_Auto(t *testing.T) {
	words, index := mustCodec(t)

	enc, err := NewEncoder(words, true)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	dec := NewDecoder(index)

	cases := [][]byte{
		[]byte(""),
		[]byte("hello world"),
		[]byte("Hello World"),
		[]byte("The quick brown fox jumps over the lazy dog."),
		[]byte{0x00, 0x01, 0x02, 0xff, 0x00, 0x10},
	}

	for _, in := range cases {
		outWords := enc.Encode(in)
		got, derr := dec.Decode(outWords, DecodeAuto)
		if derr != nil {
			// framed+auto should succeed cleanly for these
			t.Fatalf("Decode(auto) error: %v", derr)
		}
		if !bytes.Equal(got, in) {
			t.Fatalf("mismatch: got %x want %x", got, in)
		}
	}
}

func TestRoundTrip_Raw_Raw(t *testing.T) {
	words, index := mustCodec(t)

	enc, _ := NewEncoder(words, false)
	dec := NewDecoder(index)

	for n := 0; n <= 64; n++ {
		in := make([]byte, n)
		if _, err := rand.Read(in); err != nil {
			t.Fatalf("rand.Read: %v", err)
		}

		outWords := enc.Encode(in)
		got, derr := dec.Decode(outWords, DecodeRaw)
		if derr != nil {
			t.Fatalf("Decode(raw) error: %v", derr)
		}

		// RAW decode may include extra trailing 0x00 due to bit padding.
		if len(got) < len(in) {
			t.Fatalf("n=%d: decoded shorter than input: got=%d want>=%d", n, len(got), len(in))
		}
		if !bytes.Equal(got[:len(in)], in) {
			t.Fatalf("n=%d: prefix mismatch", n)
		}
		for i := len(in); i < len(got); i++ {
			if got[i] != 0x00 {
				t.Fatalf("n=%d: expected only zero padding after payload, found 0x%02x at %d", n, got[i], i)
			}
		}
	}
}

func TestAutoFallsBackToRawWhenNoMagic(t *testing.T) {
	words, index := mustCodec(t)

	encRaw, _ := NewEncoder(words, false)
	dec := NewDecoder(index)

	in := []byte("hello world")
	w := encRaw.Encode(in)

	got, derr := dec.Decode(w, DecodeAuto)
	if derr != nil {
		t.Fatalf("Decode(auto) error: %v", derr)
	}
	if !bytes.Equal(got, in) {
		t.Fatalf("mismatch: got %q want %q", string(got), string(in))
	}
}

func TestDecodeWords_SplitsWhitespace(t *testing.T) {
	words, index := mustCodec(t)

	enc, _ := NewEncoder(words, true)
	dec := NewDecoder(index)

	in := []byte("hello world")
	outWords := enc.Encode(in)

	// add newlines/tabs/multiple spaces
	s := strings.Join(outWords, "  \n\t  ")

	got, derr := dec.DecodeWords(s, DecodeAuto)
	if derr != nil {
		t.Fatalf("DecodeWords error: %v", derr)
	}
	if !bytes.Equal(got, in) {
		t.Fatalf("mismatch: got %q want %q", string(got), string(in))
	}
}
