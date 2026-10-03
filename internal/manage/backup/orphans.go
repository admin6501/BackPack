package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// A present config is enough to retain accounting, even if it is malformed.
func orphanAccounting(root, rel string) bool {
	var name string
	if !strings.Contains(rel, "/") && strings.HasSuffix(rel, ".metrics.json") {
		name = strings.TrimSuffix(rel, ".metrics.json")
	} else if strings.HasPrefix(rel, "history/") && strings.Count(rel, "/") == 1 && strings.HasSuffix(rel, ".json") {
		name = strings.TrimSuffix(strings.TrimPrefix(rel, "history/"), ".json")
	} else {
		return false
	}
	_, err := os.Lstat(filepath.Join(root, name+".toml"))
	return os.IsNotExist(err)
}

// Operates on the staging tree only, before the atomic restore commit.
func pruneOrphanAccounting(root string) error {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if !info.IsDir() && orphanAccounting(root, filepath.ToSlash(rel)) {
			return os.Remove(path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	path := filepath.Join(root, "history.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("invalid history.json: %w", err)
	}
	raw, ok := doc["tunnels"]
	if !ok {
		return nil
	}
	var tunnels map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tunnels); err != nil {
		return err
	}
	changed := false
	for name := range tunnels {
		// Never interpret a history key as a filesystem path.
		if filepath.Base(name) != name || name == "." || name == ".." {
			continue
		}
		if _, err := os.Lstat(filepath.Join(root, name+".toml")); os.IsNotExist(err) {
			delete(tunnels, name)
			changed = true
		} else if err != nil {
			return err
		}
	}
	if !changed {
		return nil
	}
	doc["tunnels"], err = json.Marshal(tunnels)
	if err != nil {
		return err
	}
	data, err = json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
