package engine

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"zhiyuwaf/internal/model"
)

type yamlRuleFile struct {
	Rules []yamlRule `yaml:"rules"`
}

type yamlRule struct {
	ID             string   `yaml:"id"`
	Name           string   `yaml:"name"`
	Description    string   `yaml:"description"`
	Severity       string   `yaml:"severity"`
	Enabled        bool     `yaml:"enabled"`
	MatchLocations []string `yaml:"match_locations"`
	Patterns       []string `yaml:"patterns"`
}

type RuleSet struct {
	rules      []Rule
	byLocation map[string][]Rule // location -> rules that check this location
	preset     string
}

func NewRuleSet() *RuleSet {
	return &RuleSet{}
}

// SetPreset configures which preset to apply when loading rules.
func (rs *RuleSet) SetPreset(name string) {
	rs.preset = name
}

func (rs *RuleSet) AddRule(r Rule) {
	rs.rules = append(rs.rules, r)
	rs.buildIndex()
}

func (rs *RuleSet) Rules() []Rule {
	return rs.rules
}

// RulesByLocation returns rules grouped by match location for optimized matching.
func (rs *RuleSet) RulesByLocation() map[string][]Rule {
	return rs.byLocation
}

func (rs *RuleSet) buildIndex() {
	idx := make(map[string][]Rule)
	for _, r := range rs.rules {
		for _, loc := range r.Locations() {
			idx[loc] = append(idx[loc], r)
		}
	}
	rs.byLocation = idx
}

func (rs *RuleSet) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("rules directory %s not found, skipping", dir)
			return nil
		}
		return fmt.Errorf("read rules dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !entry.Type().IsRegular() || strings.HasPrefix(name, ".") || filepath.Ext(name) != ".yaml" {
			continue
		}
		path := filepath.Join(dir, name)
		if err := rs.LoadFromFile(path); err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		}
	}

	log.Printf("loaded %d rules", len(rs.rules))
	rs.buildIndex()
	return nil
}

func (rs *RuleSet) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var file yamlRuleFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	// Apply preset filter if configured
	rules := file.Rules
	if rs.preset != "" {
		rules = ApplyPreset(rules, rs.preset)
	}

	for _, yr := range rules {
		if !yr.Enabled {
			continue
		}

		compiled := make([]*regexp.Regexp, 0, len(yr.Patterns))
		for _, p := range yr.Patterns {
			re, err := regexp.Compile(p)
			if err != nil {
				return fmt.Errorf("compile pattern %q in rule %s: %w", p, yr.ID, err)
			}
			compiled = append(compiled, re)
		}

		rs.rules = append(rs.rules, &PatternRule{
			BaseRule: BaseRule{
				RuleID:           yr.ID,
				RuleName:         yr.Name,
				RuleSeverity:     yr.Severity,
				CompiledPatterns: compiled,
				MatchLocations:   yr.MatchLocations,
			},
		})
	}

	return nil
}

func (rs *RuleSet) LoadFromDB(rules []model.Rule) {
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		compiled := make([]*regexp.Regexp, 0, len(r.Patterns))
		for _, p := range r.Patterns {
			re, err := regexp.Compile(p)
			if err != nil {
				log.Printf("skip invalid pattern %q in rule %s: %v", p, r.ID, err)
				continue
			}
			compiled = append(compiled, re)
		}
		rs.rules = append(rs.rules, &PatternRule{
			BaseRule: BaseRule{
				RuleID:           r.ID,
				RuleName:         r.Name,
				RuleSeverity:     r.Severity,
				CompiledPatterns: compiled,
				MatchLocations:   r.MatchLocations,
			},
		})
	}
	rs.buildIndex()
}
