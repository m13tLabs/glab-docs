package util

// A small, dependency-free port of helm.sh/helm/v3/pkg/ignore (Apache-2.0, The Helm Authors),
// trimmed to what glab-docs needs for `.glabdocsignore` handling.

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ignoreMatcher returns true when the relative path n matches the rule.
type ignoreMatcher func(n string, fi os.FileInfo) bool

type ignorePattern struct {
	raw     string
	match   ignoreMatcher
	negate  bool
	mustDir bool
}

// IgnoreRules is an ordered collection of `.gitignore`-style path matching rules.
type IgnoreRules struct {
	patterns []*ignorePattern
}

// EmptyIgnoreRules returns a ruleset that ignores nothing.
func EmptyIgnoreRules() *IgnoreRules {
	return &IgnoreRules{patterns: []*ignorePattern{}}
}

// ParseIgnoreRules reads an ignore file.
func ParseIgnoreRules(r io.Reader) (*IgnoreRules, error) {
	rules := &IgnoreRules{patterns: []*ignorePattern{}}
	scanner := bufio.NewScanner(r)
	bom := []byte{0xEF, 0xBB, 0xBF}
	first := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if first {
			line = bytes.TrimPrefix(line, bom)
			first = false
		}
		if err := rules.parseRule(string(line)); err != nil {
			return rules, err
		}
	}
	return rules, scanner.Err()
}

// Ignore evaluates path (relative to the ignore file's directory) against the rules in order and
// returns true if it should be ignored. Matching a negative rule stops evaluation.
func (r *IgnoreRules) Ignore(path string, fi os.FileInfo) bool {
	if path == "" || path == "." || path == "./" {
		return false
	}
	for _, p := range r.patterns {
		if p.match == nil {
			continue
		}
		if p.negate {
			if (p.mustDir && (fi == nil || !fi.IsDir())) || !p.match(path, fi) {
				return true
			}
			continue
		}
		if p.mustDir && (fi == nil || !fi.IsDir()) {
			continue
		}
		if p.match(path, fi) {
			return true
		}
	}
	return false
}

func (r *IgnoreRules) parseRule(rule string) error {
	rule = strings.TrimSpace(rule)
	if rule == "" || strings.HasPrefix(rule, "#") {
		return nil
	}
	if strings.Contains(rule, "**") {
		return errors.New("double-star (**) syntax is not supported")
	}
	if _, err := filepath.Match(rule, "abc"); err != nil {
		return err
	}

	p := &ignorePattern{raw: rule}
	if strings.HasPrefix(rule, "!") {
		p.negate = true
		rule = rule[1:]
	}
	if strings.HasSuffix(rule, "/") {
		p.mustDir = true
		rule = strings.TrimSuffix(rule, "/")
	}

	switch {
	case strings.HasPrefix(rule, "/"):
		rule = strings.TrimPrefix(rule, "/")
		p.match = func(n string, _ os.FileInfo) bool { return matchOrLog(rule, n) }
	case strings.Contains(rule, "/"):
		p.match = func(n string, _ os.FileInfo) bool { return matchOrLog(rule, n) }
	default:
		p.match = func(n string, _ os.FileInfo) bool { return matchOrLog(rule, filepath.Base(n)) }
	}

	r.patterns = append(r.patterns, p)
	return nil
}

func matchOrLog(pattern, name string) bool {
	ok, err := filepath.Match(pattern, name)
	if err != nil {
		log.Warnf("Failed to compile ignore pattern %q: %s", pattern, err)
		return false
	}
	return ok
}
