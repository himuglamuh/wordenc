package codec

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed wordlist_bip39_en.txt
var bip39EN []byte

func LoadBIP39English() ([]string, map[string]int, error) {
	sc := bufio.NewScanner(bytes.NewReader(bip39EN))
	words := make([]string, 0, 2048)
	for sc.Scan() {
		w := strings.TrimSpace(sc.Text())
		if w == "" {
			continue
		}
		words = append(words, w)
	}
	if err := sc.Err(); err != nil {
		return nil, nil, err
	}
	if len(words) != 2048 {
		return nil, nil, fmt.Errorf("expected 2048 words, got %d", len(words))
	}
	index := make(map[string]int, 2048)
	for i, w := range words {
		if _, ok := index[w]; ok {
			return nil, nil, fmt.Errorf("duplicate word: %q", w)
		}
		index[w] = i
	}
	return words, index, nil
}
