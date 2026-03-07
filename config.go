package repogov

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// configNames lists supported config filenames in search order.
// JSON is preferred over YAML. repogov-config is preferred over repogov.
// The dot-file form (.repogov.json) is supported for users who prefer hidden config files.
var configNames = []string{
	"repogov-config.json",
	"repogov.json",
	".repogov.json",
	"repogov-config.yaml",
	"repogov-config.yml",
	"repogov.yaml",
	"repogov.yml",
}

// LoadConfig reads a configuration file from path and returns a [Config].
// JSON and YAML formats are supported, detected by file extension
// (.yaml/.yml for YAML, everything else as JSON). Missing fields are
// filled from [DefaultConfig]. If the file does not exist, [DefaultConfig]
// is returned with a nil error, making configuration optional for callers.
func LoadConfig(path string) (Config, error) {
	def := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return def, nil
		}
		return Config{}, err
	}

	var cfg Config
	if isYAML(path) {
		if err := unmarshalYAML(data, &cfg); err != nil {
			return Config{}, err
		}
	} else {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, err
		}
	}

	// Merge missing fields from defaults.
	if cfg.Default == 0 {
		cfg.Default = def.Default
	}
	if cfg.WarningThreshold == 0 {
		cfg.WarningThreshold = def.WarningThreshold
	}
	if cfg.SkipDirs == nil {
		cfg.SkipDirs = def.SkipDirs
	}
	if cfg.Rules == nil {
		cfg.Rules = def.Rules
	}
	// Merge default file entries; user entries take precedence.
	if cfg.Files == nil {
		cfg.Files = def.Files
	} else {
		for k, v := range def.Files {
			if _, ok := cfg.Files[k]; !ok {
				cfg.Files[k] = v
			}
		}
	}

	return cfg, nil
}

// FindConfig searches for a repogov configuration file relative to root.
// It checks the repo root first, then .github/, returning the first file
// found. Within each location it tries filenames in order:
// repogov-config.json, repogov.json, repogov-config.yaml,
// repogov-config.yml, repogov.yaml, repogov.yml.
// Returns an empty string if no config file exists.
//
// Precedence: repo root over .github/, JSON over YAML,
// repogov-config over repogov.
func FindConfig(root string) string {
	dirs := []string{
		root,
		filepath.Join(root, ".github"),
	}
	for _, dir := range dirs {
		for _, name := range configNames {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

// SaveConfig writes cfg to the given path. The format is determined by
// file extension: .yaml/.yml produces YAML, everything else produces
// indented JSON. The file is created with mode 0644.
func SaveConfig(path string, cfg Config) error {
	var data []byte
	var err error
	if isYAML(path) {
		data, err = marshalYAML(cfg)
	} else {
		data, err = json.MarshalIndent(cfg, "", "  ")
		if err == nil {
			data = append(data, '\n')
		}
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Violation describes a single config validation issue.
type Violation struct {
	// Field is the config field name (e.g. "default", "rules[0].glob").
	Field string

	// Message is a human-readable description of the issue.
	Message string

	// Severity is "error" for invalid config or "warning" for suboptimal.
	Severity string
}

// ValidateConfig checks cfg for structural and semantic issues and
// returns any [Violation] entries found. An empty slice means the
// config is valid. This is intended for CLI feedback and CI pipelines.
func ValidateConfig(cfg Config) []Violation {
	var vs []Violation

	// default must be non-negative.
	if cfg.Default < 0 {
		vs = append(vs, Violation{
			Field:    "default",
			Message:  fmt.Sprintf("must be >= 0, got %d", cfg.Default),
			Severity: "error",
		})
	}

	// warning_threshold must be 0-100.
	if cfg.WarningThreshold < 0 || cfg.WarningThreshold > 100 {
		vs = append(vs, Violation{
			Field:    "warning_threshold",
			Message:  fmt.Sprintf("must be 0-100, got %d", cfg.WarningThreshold),
			Severity: "error",
		})
	}

	// Rules validation.
	for i, r := range cfg.Rules {
		field := fmt.Sprintf("rules[%d]", i)
		if r.Glob == "" {
			vs = append(vs, Violation{
				Field:    field + ".glob",
				Message:  "glob pattern is empty",
				Severity: "error",
			})
		} else if _, err := filepath.Match(r.Glob, "test"); err != nil {
			vs = append(vs, Violation{
				Field:    field + ".glob",
				Message:  fmt.Sprintf("invalid glob pattern %q: %v", r.Glob, err),
				Severity: "error",
			})
		}
		if r.Limit < 0 {
			vs = append(vs, Violation{
				Field:    field + ".limit",
				Message:  fmt.Sprintf("must be >= 0, got %d", r.Limit),
				Severity: "error",
			})
		}
	}

	// Files validation.
	for path, limit := range cfg.Files {
		field := fmt.Sprintf("files[%q]", path)
		if path == "" {
			vs = append(vs, Violation{
				Field:    "files",
				Message:  "empty file path key",
				Severity: "error",
			})
		}
		if limit < 0 {
			vs = append(vs, Violation{
				Field:    field,
				Message:  fmt.Sprintf("must be >= 0, got %d", limit),
				Severity: "error",
			})
		}
		// Warn on backslashes (should use forward slashes).
		if strings.Contains(path, "\\") {
			vs = append(vs, Violation{
				Field:    field,
				Message:  "use forward slashes in file paths",
				Severity: "warning",
			})
		}
	}

	// Warn on duplicate glob patterns.
	seen := make(map[string]int)
	for i, r := range cfg.Rules {
		if prev, ok := seen[r.Glob]; ok {
			vs = append(vs, Violation{
				Field:    fmt.Sprintf("rules[%d].glob", i),
				Message:  fmt.Sprintf("duplicate glob %q (first at rules[%d])", r.Glob, prev),
				Severity: "warning",
			})
		}
		seen[r.Glob] = i
	}

	// Warn on files with a less-restrictive limit than a matching rule.
	// A stricter per-file limit (lower or zero/exempt) is fine, but a
	// higher limit may indicate a misconfiguration.
	for path, limit := range cfg.Files {
		for _, r := range cfg.Rules {
			if ok, _ := filepath.Match(r.Glob, path); ok {
				if limit > r.Limit && r.Limit > 0 {
					vs = append(vs, Violation{
						Field:    fmt.Sprintf("files[%q]", path),
						Message:  fmt.Sprintf("limit %d exceeds rule %q (limit %d); per-file entry takes precedence", limit, r.Glob, r.Limit),
						Severity: "warning",
					})
				}
				break
			}
		}
	}

	return vs
}
