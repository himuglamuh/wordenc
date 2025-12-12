package main

import (
	"bytes"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildWordenc(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	// repo root is ../..
	root := filepath.Clean(filepath.Join(wd, "..", ".."))

	bin := filepath.Join(t.TempDir(), "wordenc-testbin")

	cmd := exec.Command("go", "build", "-o", bin, "./cmd/wordenc")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(out))
	}

	return bin
}

func runCmd(t *testing.T, bin string, stdin []byte, args ...string) (stdout, stderr []byte, err error) {
	t.Helper()

	cmd := exec.Command(bin, args...)
	cmd.Stdin = bytes.NewReader(stdin)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

func TestCLI_RoundTrip_Framed_Stdin(t *testing.T) {
	bin := buildWordenc(t)

	in := []byte("hello world\n")

	encOut, encErr, err := runCmd(t, bin, in, "encode")
	if err != nil {
		t.Fatalf("encode failed: %v\nstderr=%s", err, encErr)
	}

	decOut, decErr, err := runCmd(t, bin, encOut, "decode")
	if err != nil {
		t.Fatalf("decode failed: %v\nstderr=%s", err, decErr)
	}

	// CLI prints UTF-8 as text; ensure exact.
	if !bytes.Equal(decOut, in) {
		t.Fatalf("roundtrip mismatch: got=%q want=%q", string(decOut), string(in))
	}
}

func TestCLI_RoundTrip_Framed_FileIO(t *testing.T) {
	bin := buildWordenc(t)

	tmp := t.TempDir()
	inFile := filepath.Join(tmp, "in.bin")
	wordsFile := filepath.Join(tmp, "out.words")
	outFile := filepath.Join(tmp, "out.bin")

	// some binary including zeros/newlines
	in := []byte{0x00, 0x01, 0x02, 0xff, 0x00, 0x10, '\n', 'X', 0x00}
	if err := os.WriteFile(inFile, in, 0o644); err != nil {
		t.Fatalf("WriteFile inFile: %v", err)
	}

	encOut, encErr, err := runCmd(t, bin, nil, "encode", "-in", inFile)
	if err != nil {
		t.Fatalf("encode -in failed: %v\nstderr=%s", err, encErr)
	}
	if err := os.WriteFile(wordsFile, encOut, 0o644); err != nil {
		t.Fatalf("WriteFile wordsFile: %v", err)
	}

	_, decErr, err := runCmd(t, bin, nil, "decode", "-in", wordsFile, "-out", outFile)
	if err != nil {
		t.Fatalf("decode -out failed: %v\nstderr=%s", err, decErr)
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile outFile: %v", err)
	}
	if !bytes.Equal(got, in) {
		t.Fatalf("file roundtrip mismatch: got=%x want=%x", got, in)
	}
}

func TestCLI_RoundTrip_Raw_WithHex(t *testing.T) {
	bin := buildWordenc(t)

	// raw mode may pad with extra zeros on decode
	// using --hex makes it easy to assert at least a prefix match
	// use framed here to assert exact, and raw to assert prefix
	in := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x01}

	// encode raw
	encOut, encErr, err := runCmd(t, bin, in, "encode", "--raw")
	if err != nil {
		t.Fatalf("encode --raw failed: %v\nstderr=%s", err, encErr)
	}

	// decode raw to hex
	decHex, decErr, err := runCmd(t, bin, encOut, "decode", "--mode", "raw", "--hex")
	if err != nil {
		t.Fatalf("decode --mode raw --hex failed: %v\nstderr=%s", err, decErr)
	}

	gotHex := strings.TrimSpace(string(decHex))
	got, err := hex.DecodeString(gotHex)
	if err != nil {
		t.Fatalf("hex decode failed: %v (hex=%q)", err, gotHex)
	}

	// raw decode must start with original bytes
	// may include trailing 0x00 padding
	if len(got) < len(in) {
		t.Fatalf("raw decoded shorter: got=%d want>=%d", len(got), len(in))
	}
	if !bytes.Equal(got[:len(in)], in) {
		t.Fatalf("raw prefix mismatch: got=%x wantprefix=%x", got[:len(in)], in)
	}
	for i := len(in); i < len(got); i++ {
		if got[i] != 0x00 {
			t.Fatalf("expected only 0x00 padding after payload, found 0x%02x at %d", got[i], i)
		}
	}
}
