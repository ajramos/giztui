package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// obsoleteKeyPaths lists dotted paths that were removed from the config schema. Migration prunes
// them from existing user files so stale, no-longer-honored options don't linger and mislead.
var obsoleteKeyPaths = []string{
	"keys.prompt_test", // never had a handler; removed in v1.11.2
}

// pruneObsolete deletes each dotted path in paths from user (descending into nested maps).
// Returns the paths actually removed.
func pruneObsolete(user map[string]any, paths []string) []string {
	var removed []string
	for _, p := range paths {
		parts := strings.Split(p, ".")
		m := user
		ok := true
		for _, seg := range parts[:len(parts)-1] {
			next, isMap := m[seg].(map[string]any)
			if !isMap {
				ok = false
				break
			}
			m = next
		}
		if !ok {
			continue
		}
		leaf := parts[len(parts)-1]
		if _, exists := m[leaf]; exists {
			delete(m, leaf)
			removed = append(removed, p)
		}
	}
	return removed
}

// deepMergeMissing adds into user every key present in defaults but absent from user, recursing
// into nested objects. It never overwrites an existing user value. Returns the dotted paths added.
func deepMergeMissing(user, defaults map[string]any, prefix string) []string {
	var added []string
	for k, dv := range defaults {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		uv, ok := user[k]
		if !ok {
			user[k] = dv
			added = append(added, path)
			continue
		}
		if dvMap, dok := dv.(map[string]any); dok {
			if uvMap, uok := uv.(map[string]any); uok {
				added = append(added, deepMergeMissing(uvMap, dvMap, path)...)
			}
		}
		// present scalar / type mismatch → keep the user's value.
	}
	return added
}

// defaultConfigMap returns DefaultConfig() as a generic map.
func defaultConfigMap() (map[string]any, error) {
	data, err := json.Marshal(DefaultConfig())
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// readConfigMap reads a config file as a generic map (preserving _comment keys). A missing or
// empty file yields an empty map; invalid JSON is an error.
func readConfigMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("config is not valid JSON: %w", err)
	}
	return m, nil
}

// MissingDefaultKeys returns the dotted paths of default keys absent from the user's config file.
// Read-only (used for the startup notice).
func MissingDefaultKeys(path string) ([]string, error) {
	defaults, err := defaultConfigMap()
	if err != nil {
		return nil, err
	}
	user, err := readConfigMap(path)
	if err != nil {
		return nil, err
	}
	return deepMergeMissing(user, defaults, ""), nil // mutates the local user map; discarded
}

// MigrateConfigFile reconciles the user's config file with the current schema, writing a .bak
// first: it adds missing default keys and prunes obsolete ones (see obsoleteKeyPaths). Returns the
// added and removed dotted paths and the backup path. No-op (nil, nil, "", nil) when there is
// nothing to add or remove. Output is json.MarshalIndent (2-space), keys alphabetically sorted.
func MigrateConfigFile(path string) (added, removed []string, backupPath string, err error) {
	path = filepath.Clean(path)
	defaults, err := defaultConfigMap()
	if err != nil {
		return nil, nil, "", err
	}
	user, err := readConfigMap(path)
	if err != nil {
		return nil, nil, "", err
	}
	added = deepMergeMissing(user, defaults, "")
	removed = pruneObsolete(user, obsoleteKeyPaths)
	if len(added) == 0 && len(removed) == 0 {
		return nil, nil, "", nil
	}
	out, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return nil, nil, "", err
	}
	if orig, rerr := os.ReadFile(path); rerr == nil { // #nosec G304 -- path is filepath.Clean'd above
		backupPath = path + ".bak"
		// #nosec G703 -- backupPath is the cleaned config path with a fixed .bak suffix
		if werr := os.WriteFile(backupPath, orig, 0600); werr != nil {
			return nil, nil, "", fmt.Errorf("could not write backup %s: %w", backupPath, werr)
		}
	}
	if werr := os.WriteFile(path, out, 0600); werr != nil {
		return nil, nil, "", werr
	}
	return added, removed, backupPath, nil
}
