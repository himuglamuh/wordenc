# wordenc

**wordenc** is a reversible encoder/decoder that maps arbitrary bytes to a sequence of human-readable words using the BIP-39 English wordlist (2048 words, 11 bits per word).

It supports two modes:

* **Raw** — pure word <-> byte encoding, no framing, maximum portability
* **Framed** — self-describing encoding with a header and length prefix, so padding is handled automatically

It is usable both as:

* a **CLI tool** (`wordenc encode`, `wordenc decode`)
* a **Go module** for embedding in other programs

This project was designed with ARGs, steganography, and human-in-the-loop workflows in mind. It is not made for rigid protocols.

## Features

* Uses the **standard BIP-39 English wordlist** (2048 words)
* Reversible encoding for **arbitrary binary data**
* Optional **length framing** (no checksums, no crypto)
* Automatic decode with graceful fallback
* Handles truncated input and extra padding sensibly
* Zero external dependencies
* Library and CLI share the same implementation

## Encoding overview

### Word mapping

* Each word represents **11 bits**
* Bytes are packed MSB-first into 11-bit indices
* Final word is zero-padded if needed

### Modes

#### Raw mode

```
payload > bits > 11-bit words
```

* Smallest possible output
* No length information
* Decoder cannot know where padding begins
* Most portable (other decoders can handle it)

#### Framed mode

```
MAGIC (8 bytes) + length (uvarint) + payload > words
```

* Decoder knows exact payload length
* Padding handled automatically
* Slightly larger output
* Intended for use when both sides use `wordenc`

---

## Framed header format

The framed encoding starts with a fixed **8-byte header**:

```
"WENC" 0x01 0x00 0xA7 '|'
```

| Byte(s) | Meaning                            |
| ------- | ---------------------------------- |
| 0–3     | ASCII `"WENC"` identifier          |
| 4       | Version (currently `0x01`)         |
| 5       | Flags (reserved, currently `0x00`) |
| 6       | Poison byte (`0xA7`)               |
| 7       | Human-visible delimiter (`|`)      |

This header is:

* **extremely unlikely** to appear in raw data by accident
* obvious if someone decodes framed data with a raw-only decoder
* versioned and future-proof

Immediately after the header comes an **unsigned varint payload length**, followed by the payload bytes.

## Decode behavior (important)

By default, decoding uses **Auto** mode:

1. Decode words -> bytes (best effort)
2. If bytes start with the framed header:
   * parse length
   * if length fits -> return exactly that many bytes
   * if length exceeds available data -> return partial payload
3. If header is missing or implausible -> fall back to Raw output

This means:

* wrong mode != hard failure
* truncated input still produces useful output
* humans aren’t punished for copy/paste mistakes

Strict behavior is available if you want it.

## CLI usage

### Encode

```bash
# Encode stdin (framed by default)
echo "hello world" | wordenc encode

# Encode raw (no header, max portability)
echo "hello world" | wordenc encode --raw

# Encode a binary file
wordenc encode -in image.png > image.words
```

### Decode

```bash
# Auto-detect framed vs raw
wordenc decode < words.txt

# Force raw decode
wordenc decode --mode raw < words.txt

# Force framed decode
wordenc decode --mode framed < words.txt

# Write output to file
wordenc decode -in image.words -out image.png
```

### Useful flags

| Flag                       | Description                                |
| -------------------------- | ------------------------------------------ |
| `--raw`                    | Encode without header                      |
| `--mode auto\|raw\|framed` | Decode mode                                |
| `--strict`                 | Error on truncation/bad padding            |
| `--hex`                    | Print decoded bytes as hex                 |
| `--maxlenmb`               | Max framed payload length (default 256 MB) |
| `-in`, `-out`              | Input/output files (default stdin/stdout)  |

## Go module usage

### Install

```bash
go get github.com/himuglamuh/wordenc
```

### Encoding

```go
words, _, _ := codec.LoadBIP39English()

enc, _ := codec.NewEncoder(words, true) // framed
out := enc.Encode([]byte("hello world"))

fmt.Println(strings.Join(out, " "))
```

### Decoding

```go
_, index, _ := codec.LoadBIP39English()

dec := codec.NewDecoder(index)
dec.MaxLen = 512 << 20 // 512 MB
dec.AllowPartial = true

data, err := dec.DecodeWords(wordString, codec.DecodeAuto)
```

### Decode modes

```go
codec.DecodeAuto    // default, recommended
codec.DecodeRaw
codec.DecodeFramed
```

## Error philosophy

This tool is **forgiving by default**.

* Truncated input returns partial output
* Extra words after payload are ignored
* Header collisions fall back to raw mode
* CLI prints warnings instead of failing

If you want correctness over convenience, use `--strict` or configure the decoder accordingly.

## Length limits

* Length is encoded as an **unsigned varint (`uint64`)**
* Theoretical maximum: **2⁶⁴ − 1 bytes (~18 exabytes)**
* Practical limits are enforced via:
  * decoder `MaxLen`
  * available memory
  * Go slice limits

Raw mode has no length information.

## Interoperability notes

* Raw mode is compatible with any decoder that:
  * uses the BIP-39 wordlist
  * treats words as base-2048 symbols
* Framed mode is specific to `wordenc`
* If framed data is decoded as raw, the header appears as harmless junk at the front of the output

## Non-goals

This project does **not** aim to be:

* a cryptographic scheme
* a compression format
* a secure transport protocol
* a strict standard

It is a **tool**.

## License

MIT. Do weird things with it.
