package codec

import "errors"

var (
	ErrBadWord      = errors.New("bad word: not in wordlist")
	ErrBadPadding   = errors.New("non-zero padding bits")
	ErrBadHeader    = errors.New("invalid header")
	ErrTruncated    = errors.New("truncated input")
	ErrAbsurdLength = errors.New("declared length exceeds configured maximum")
)
