// Command signsums signs a release's SHA256SUMS, for the release workflow.
//
// It reads the private key from RELEASE_SIGNING_KEY — base64 of the 64-byte
// Ed25519 private key that `make release-key` printed — and writes a detached
// signature beside the file. The signature is base64 of the raw 64 bytes, which
// is what the updater expects; see internal/manage/releasesig.go.
//
// With no key in the environment it does nothing and says so, so a fork or a
// local `make release` still produces a full set of assets.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"github.com/backpack/backpack/internal/app"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintln(os.Stderr, "usage: signsums <path to SHA256SUMS> [tag] | --public-key")
		os.Exit(2)
	}
	path := os.Args[1]
	key, err := signingKey(os.Getenv("RELEASE_SIGNING_KEY"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if path == "--public-key" {
		if key != nil {
			fmt.Println(base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey)))
		}
		return
	}
	// The tag is signed with the list: see app.ReleaseSignedMessage. On CI it
	// is the tag being released; by hand, the one given, or the VERSION file.
	tag := releaseTag()
	if len(os.Args) == 3 {
		tag = os.Args[2]
	}
	if tag == "" {
		fmt.Fprintln(os.Stderr, "no release tag: set GITHUB_REF_NAME, pass one, or run from the repository root")
		os.Exit(1)
	}

	if key == nil {
		if err := os.Remove(path + ".sig"); err != nil && !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("No signing key configured; release uses SHA256 checksums only.")
		return
	}

	sums, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sig := ed25519.Sign(key, app.ReleaseSignedMessage(tag, sums))
	out := path + ".sig"
	if err := os.WriteFile(out, []byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Signed for", tag+":", out)
}

// releaseTag is the tag being released: CI's, or the VERSION file's.
func releaseTag() string {
	if t := strings.TrimSpace(os.Getenv("GITHUB_REF_NAME")); t != "" {
		return t
	}
	v, err := os.ReadFile("VERSION")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(v))
}
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
