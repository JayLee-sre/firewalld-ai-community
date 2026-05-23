package engine

// Preset defines which rules to enable/disabled and how aggressively to match.
type Preset struct {
	Name        string
	Description string
	DisabledIDs map[string]bool // rule IDs to disable
}

var presets = map[string]Preset{
	"strict": {
		Name:        "strict",
		Description: "所有规则启用，严格防护（可能有较多误报）",
		DisabledIDs: map[string]bool{},
	},
	"balanced": {
		Name:        "balanced",
		Description: "平衡模式，跳过过于激进的规则（推荐）",
		DisabledIDs: map[string]bool{
			"NOSQL-001": true,
			"SCAN-001":  true,
			"BOT-001":   true, // matches any login form
		},
	},
	"permissive": {
		Name:        "permissive",
		Description: "宽松模式，仅启用高危规则",
		DisabledIDs: map[string]bool{
			"SCAN-001":      true,
			"DISCOVERY-001": true,
			"BOT-001":       true,
			"NOSQL-001":     true,
			"SQLI-002":      true,
			"SQLI-003":      true,
			"SSRF-001":      true,
			"WEBSHELL-001":  true,
			"SENSITIVE-001": true,
			"SENSITIVE-002": true,
			"SENSITIVE-003": true,
		},
	},
}

// GetPreset returns a preset by name. Returns balanced if not found.
func GetPreset(name string) Preset {
	if p, ok := presets[name]; ok {
		return p
	}
	return presets["balanced"]
}

// PresetNames returns all available preset names.
func PresetNames() []string {
	return []string{"strict", "balanced", "permissive"}
}

// ApplyPreset filters a yamlRule slice based on the preset.
// It disables rules listed in the preset's DisabledIDs.
func ApplyPreset(rules []yamlRule, presetName string) []yamlRule {
	preset := GetPreset(presetName)
	if preset.Name == "strict" {
		return rules // enable everything
	}

	var filtered []yamlRule
	for _, r := range rules {
		if preset.DisabledIDs[r.ID] {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}
