// Command signsums signs a release's SHA256SUMS, for the release workflow.
//
// It reads the private key from RELEASE_SIGNING_KEY — base64 of the 64-byte
// Ed25519 private key that `make release-key` printed — and writes a detached
// signature beside the file. The signature is base64 of the raw 64 bytes, which
// is what the updater expects; see internal/manage/releasesig.go.
//
// Signing is optional when no key is configured. Invalid configured keys are
// errors. --public-key prints only the public half for embedding in binaries.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: signsums <path to SHA256SUMS> | --public-key")
		os.Exit(2)
	}
	path := os.Args[1]
	key, err := signingKey(os.Getenv("RELEASE_SIGNING_KEY"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if key == nil {
		if path != "--public-key" {
			// Do not leave an earlier build's signature beside new checksums.
			if err := os.Remove(path + ".sig"); err != nil && !os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("No signing key configured; release uses SHA256 checksums only.")
		}
		return
	}
	if path == "--public-key" {
		fmt.Println(base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey)))
		return
	}

	sums, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sig := ed25519.Sign(ed25519.PrivateKey(key), sums)
	out := path + ".sig"
	if err := os.WriteFile(out, []byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Signed:", out)
}

// Regenerate the private key from its seed to reject inconsistent public halves.
func signingKey(encoded string) (ed25519.PrivateKey, error) {
	if strings.TrimSpace(encoded) == "" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("RELEASE_SIGNING_KEY must contain a base64 Ed25519 private key")
	}
	key := ed25519.NewKeyFromSeed(raw[:ed25519.SeedSize])
	if !key.Equal(ed25519.PrivateKey(raw)) {
		return nil, fmt.Errorf("RELEASE_SIGNING_KEY has an inconsistent public key")
	}
	return key, nil
}
