package node

import (
	"bytes"
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

func TestSavingCredentialsPreservesADamagedFleetKeyAndRegistry(t *testing.T) {
	isolateStore(t)
	original := Store{Nodes: []Node{{Name: "existing", Password: "saved-password"}}}
	if err := SaveStore(original); err != nil {
		t.Fatal(err)
	}
	registry, err := os.ReadFile(StorePath)
	if err != nil {
		t.Fatal(err)
	}
	damaged := []byte("truncated-key")
	if err := os.WriteFile(keyPath(), damaged, 0600); err != nil {
		t.Fatal(err)
	}
	if err := SaveStore(original); err == nil {
		t.Fatal("saving with a damaged fleet key succeeded; the key was silently replaced")
	}
	if got, err := os.ReadFile(keyPath()); err != nil || !bytes.Equal(got, damaged) {
		t.Fatalf("damaged key was changed: %v", err)
	}
	if got, err := os.ReadFile(StorePath); err != nil || !bytes.Equal(got, registry) {
		t.Fatalf("registry was changed after a key error: %v", err)
	}
}

func TestSavingCredentialsDoesNotReplaceAnUnreadableFleetKey(t *testing.T) {
	isolateStore(t)
	// A symlink loop gives a real read error even when the tests run as root.
	if err := os.Symlink(sealKeyName, keyPath()); err != nil {
		t.Fatal(err)
	}
	if _, err := seal("new-password"); err == nil || !strings.Contains(err.Error(), "reading the fleet key") {
		t.Fatalf("expected the key read error, got %v", err)
	}
	if target, err := os.Readlink(keyPath()); err != nil || target != sealKeyName {
		t.Fatalf("unreadable key was replaced: %v", err)
	}
}

func TestImportingAFleetKeyPreservesExistingUnreadableOrEmptyKeys(t *testing.T) {
	for _, kind := range []string{"empty", "unreadable"} {
		t.Run(kind, func(t *testing.T) {
			isolateStore(t)
			if kind == "empty" {
				if err := os.WriteFile(keyPath(), nil, 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Symlink(sealKeyName, keyPath()); err != nil {
					t.Fatal(err)
				}
			}
			key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
			if err := ImportSealKey(key); err == nil {
				t.Fatal("import replaced an existing key instead of refusing")
			}
			info, err := os.Lstat(keyPath())
			if err != nil {
				t.Fatal(err)
			}
			if kind == "empty" && info.Size() != 0 {
				t.Fatal("empty key was overwritten")
			}
			if kind == "unreadable" && info.Mode()&os.ModeSymlink == 0 {
				t.Fatal("unreadable key was overwritten")
			}
		})
	}
}
