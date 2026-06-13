package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

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
	data, err := os.ReadFile(path)
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

// MigrateConfigFile adds missing default keys to the user's config file, writing a .bak first.
// Returns the added dotted paths and the backup path. No-op (nil, "", nil) when nothing is missing.
// Output is json.MarshalIndent (2-space), keys alphabetically sorted.
func MigrateConfigFile(path string) ([]string, string, error) {
	defaults, err := defaultConfigMap()
	if err != nil {
		return nil, "", err
	}
	user, err := readConfigMap(path)
	if err != nil {
		return nil, "", err
	}
	added := deepMergeMissing(user, defaults, "")
	if len(added) == 0 {
		return nil, "", nil
	}
	out, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return nil, "", err
	}
	backupPath := ""
	if orig, rerr := os.ReadFile(path); rerr == nil {
		backupPath = path + ".bak"
		if werr := os.WriteFile(backupPath, orig, 0600); werr != nil {
			return nil, "", fmt.Errorf("could not write backup %s: %w", backupPath, werr)
		}
	}
	if err := os.WriteFile(path, out, 0600); err != nil {
		return nil, "", err
	}
	return added, backupPath, nil
}
