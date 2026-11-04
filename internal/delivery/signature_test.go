
package delivery

import (
	"strings"
	"testing"
)

func TestGenerateSignature(t *testing.T) {
	payload := []byte(`{"payment_id":12345,"amount":5000}`)
	secret := "acme-secret"

	signature := GenerateSignature(payload, secret)

	if !strings.HasPrefix(signature, "sha256=") {
		t.Fatal("signature must start with sha256=")
	}

	if len(signature) != 71 {
		t.Fatalf("expected signature length 71, got %d", len(signature))
	}

	// The same payload and secret must produce the same signature.
	secondSignature := GenerateSignature(payload, secret)

	if signature != secondSignature {
		t.Fatal("signatures should be identical for the same input")
	}

	// A different secret must produce a different signature.
	differentSignature := GenerateSignature(payload, "wrong-secret")

	if signature == differentSignature {
		t.Fatal("different secrets must produce different signatures")
	}
}
