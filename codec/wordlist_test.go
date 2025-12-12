package codec

import "testing"

func TestLoadBIP39English(t *testing.T) {
	words, index, err := LoadBIP39English()
	if err != nil {
		t.Fatalf("LoadBIP39English() error: %v", err)
	}
	if len(words) != 2048 {
		t.Fatalf("expected 2048 words, got %d", len(words))
	}
	if len(index) != 2048 {
		t.Fatalf("expected 2048 index entries, got %d", len(index))
	}
	// quick spot-check: every word must map back to its index
	for i, w := range words {
		if idx, ok := index[w]; !ok || idx != i {
			t.Fatalf("index mismatch for %q: ok=%v idx=%d want=%d", w, ok, idx, i)
		}
	}
}
