package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestSigningKeyMatchesPublisher(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := signingKey(base64.StdEncoding.EncodeToString(priv))
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Public().(ed25519.PublicKey).Equal(pub) {
		t.Fatal("public key changed")
	}
	if !ed25519.Verify(pub, []byte("sums"), ed25519.Sign(parsed, []byte("sums"))) {
		t.Fatal("signature did not verify")
	}
	priv[len(priv)-1] ^= 1
	for _, bad := range []string{"", "garbage", base64.StdEncoding.EncodeToString(priv)} {
		if _, err := signingKey(bad); err == nil {
			t.Fatal("invalid release key accepted")
		}
	}
}
