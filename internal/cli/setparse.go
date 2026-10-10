package cli

import (
	"fmt"
	"strings"
)

// parseSetFlags parses Helm-style --set flags: repeatable, comma-separated
// groups of k=v, supporting {a,b} list values (for multichoice) and \, \= \\ escapes.
func parseSetFlags(values []string) (map[string]any, error) {
	out := map[string]any{}
	for _, s := range values {
		for _, pair := range splitSet(s) {
			if pair == "" {
				continue
			}
			k, v, err := splitKV(pair)
			if err != nil {
				return nil, fmt.Errorf("--set %q: %w", s, err)
			}
			if k == "" {
				return nil, fmt.Errorf("--set %q: empty key", s)
			}
			out[k] = v
		}
	}
	return out, nil
}

// splitSet splits on commas that are neither escaped nor inside {}.
func splitSet(s string) []string {
	var parts []string
	var buf strings.Builder
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			buf.WriteByte(c)
			buf.WriteByte(s[i+1])
			i++
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, buf.String())
				buf.Reset()
				continue
			}
		}
		buf.WriteByte(c)
	}
	parts = append(parts, buf.String())
	return parts
}

// splitKV splits key and value on the first unescaped =.
func splitKV(s string) (string, any, error) {
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '=' {
			return unescape(s[:i]), parseValue(s[i+1:]), nil
		}
	}
	return "", nil, fmt.Errorf("missing '='")
}

// parseValue parses a value: {a,b} is a list, anything else is a string.
func parseValue(v string) any {
	if strings.HasPrefix(v, "{") && strings.HasSuffix(v, "}") {
		inner := v[1 : len(v)-1]
		if inner == "" {
			return []string{}
		}
		parts := splitSet(inner)
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			out = append(out, unescape(p))
		}
		return out
	}
	return unescape(v)
}

// unescape resolves \x escape sequences.
func unescape(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
