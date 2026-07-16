// Package redactor provides in-memory PII redaction for statement uploads.
//
// Slice 1 uses regex/pattern matching only (IBAN, SSN-like, long digit runs,
// phone numbers). It does not perform NLP named-entity recognition for personal
// names. Merchant billing descriptors are intentionally preserved.
package redactor

import (
	"bytes"
	"io"
	"regexp"
)

const maxBytes = 20 << 20 // 20 MiB

var (
	reIBAN = regexp.MustCompile(`(?i)\b[A-Z]{2}\d{2}[A-Z0-9]{11,30}\b`)
	reSSN  = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	// US-style phones with optional country code / separators.
	rePhone = regexp.MustCompile(`(?i)(?:\+?1[\s\-.]?)?(?:\(?\d{3}\)?[\s\-.]?)\d{3}[\s\-.]?\d{4}\b`)
	// Long digit runs that look like account numbers (12–19 digits, not part of IBAN already replaced).
	reAccount = regexp.MustCompile(`\b\d{12,19}\b`)
)

// Bytes redacts PII patterns in b and returns a new slice.
// Order matters: long digit runs (accounts) before phone patterns so bare
// 10+ digit sequences are not partially matched as phone numbers.
func Bytes(b []byte) []byte {
	out := reIBAN.ReplaceAll(b, []byte("[REDACTED_IBAN]"))
	out = reSSN.ReplaceAll(out, []byte("[REDACTED_SSN]"))
	out = reAccount.ReplaceAll(out, []byte("[REDACTED_ACCOUNT]"))
	out = rePhone.ReplaceAll(out, []byte("[REDACTED_PHONE]"))
	return out
}

// Reader reads all of r (up to maxBytes), redacts, and returns the result.
func Reader(r io.Reader) ([]byte, error) {
	limited := io.LimitReader(r, int64(maxBytes)+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(raw) > maxBytes {
		return nil, ErrTooLarge
	}
	return Bytes(raw), nil
}

// ErrTooLarge is returned when the input exceeds the 20MB upload limit.
var ErrTooLarge = errTooLarge{}

type errTooLarge struct{}

func (errTooLarge) Error() string { return "upload exceeds 20MB limit" }

// ContainsToken reports whether redacted output still contains a raw needle
// (used in tests to ensure PII was removed).
func ContainsToken(redacted []byte, token string) bool {
	return bytes.Contains(redacted, []byte(token))
}
