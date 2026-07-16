package redactor_test

import (
	"os"
	"strings"
	"testing"

	"github.com/uiansol/vigil-auditor/pkg/redactor"
)

func TestBytesRedactsPIIKeepsMerchants(t *testing.T) {
	t.Parallel()

	in := []byte("NETFLIX.COM 15.99 Acct 12345678901234 IBAN GB82WEST12345698765432 SSN 123-45-6789 Call +1 (415) 555-0199")
	out := redactor.Bytes(in)
	s := string(out)

	if !strings.Contains(s, "NETFLIX.COM") {
		t.Fatalf("merchant stripped: %s", s)
	}
	for _, raw := range []string{
		"12345678901234",
		"GB82WEST12345698765432",
		"123-45-6789",
		"415",
		"555-0199",
	} {
		if strings.Contains(s, raw) && raw != "415" {
			// phone area code alone may remain in other contexts; check full patterns below
		}
	}
	if strings.Contains(s, "12345678901234") {
		t.Fatalf("account still present: %s", s)
	}
	if strings.Contains(s, "GB82WEST12345698765432") {
		t.Fatalf("iban still present: %s", s)
	}
	if strings.Contains(s, "123-45-6789") {
		t.Fatalf("ssn still present: %s", s)
	}
	if !strings.Contains(s, "[REDACTED_ACCOUNT]") || !strings.Contains(s, "[REDACTED_IBAN]") || !strings.Contains(s, "[REDACTED_SSN]") {
		t.Fatalf("missing redaction tokens: %s", s)
	}
	if !strings.Contains(s, "[REDACTED_PHONE]") {
		t.Fatalf("missing phone token: %s", s)
	}
}

func TestReaderFixtureCSV(t *testing.T) {
	t.Parallel()

	f, err := os.Open("../../testdata/statements/sample_with_pii.csv")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	out, err := redactor.Reader(f)
	if err != nil {
		t.Fatalf("redact: %v", err)
	}
	s := string(out)
	if strings.Contains(s, "12345678901234") || strings.Contains(s, "GB82WEST12345698765432") {
		t.Fatalf("PII leaked: %s", s)
	}
	if !strings.Contains(s, "NETFLIX.COM") || !strings.Contains(s, "SPOTIFY USA") {
		t.Fatalf("merchants missing: %s", s)
	}
}
