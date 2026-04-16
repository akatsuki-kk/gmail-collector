package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Search  SearchConfig      `yaml:"search"`
	Extract map[string]string `yaml:"extract"`
	Output  OutputConfig      `yaml:"output"`
}

type SearchConfig struct {
	From             []string `yaml:"from"`
	SubjectContains  []string `yaml:"subject_contains"`
	BodyContains     []string `yaml:"body_contains"`
	After            string   `yaml:"after"`
	Before           string   `yaml:"before"`
	Label            []string `yaml:"label"`
	IncludeSpamTrash bool     `yaml:"include_spam_trash"`
}

type OutputConfig struct {
	Pretty bool `yaml:"pretty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c Config) Validate() error {
	if len(c.Extract) == 0 {
		return fmt.Errorf("extract must contain at least one rule")
	}

	for key, pattern := range c.Extract {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("extract key must not be empty")
		}
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("extract rule %q must not be empty", key)
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("extract rule %q is invalid: %w", key, err)
		}
	}

	if len(c.Search.From) == 0 && len(c.Search.SubjectContains) == 0 && len(c.Search.BodyContains) == 0 && c.Search.After == "" && c.Search.Before == "" && len(c.Search.Label) == 0 {
		return fmt.Errorf("search must include at least one condition")
	}

	return nil
}

func (s SearchConfig) BuildQuery() string {
	if len(s.SubjectContains) == 0 {
		return s.buildQueryWithSubject("")
	}
	return s.buildQueryWithSubject(s.SubjectContains[0])
}

func (s SearchConfig) BuildQueries() []string {
	if len(s.SubjectContains) == 0 {
		return []string{s.buildQueryWithSubject("")}
	}

	queries := make([]string, 0, len(s.SubjectContains))
	for _, subject := range s.SubjectContains {
		queries = append(queries, s.buildQueryWithSubject(subject))
	}

	return queries
}

func (s SearchConfig) buildQueryWithSubject(subjectFilter string) string {
	var parts []string

	for _, from := range s.From {
		if trimmed := strings.TrimSpace(from); trimmed != "" {
			parts = append(parts, fmt.Sprintf("from:%s", quoteIfNeeded(trimmed)))
		}
	}
	if trimmed := strings.TrimSpace(subjectFilter); trimmed != "" {
		parts = append(parts, fmt.Sprintf("subject:%s", quoteIfNeeded(trimmed)))
	}
	for _, body := range s.BodyContains {
		if trimmed := strings.TrimSpace(body); trimmed != "" {
			parts = append(parts, quoteIfNeeded(trimmed))
		}
	}
	if s.After != "" {
		parts = append(parts, fmt.Sprintf("after:%s", s.After))
	}
	if s.Before != "" {
		parts = append(parts, fmt.Sprintf("before:%s", s.Before))
	}
	for _, label := range s.Label {
		if trimmed := strings.TrimSpace(label); trimmed != "" {
			parts = append(parts, fmt.Sprintf("label:%s", quoteIfNeeded(trimmed)))
		}
	}

	return strings.Join(parts, " ")
}

func quoteIfNeeded(value string) string {
	if strings.ContainsAny(value, " \t") {
		return fmt.Sprintf("%q", value)
	}
	return value
}
