package extract

import (
	"fmt"
	"regexp"
	"sort"
)

func Apply(body string, rules map[string]string) (map[string]string, error) {
	keys := sortedKeys(rules)
	result := make(map[string]string, len(keys))

	for _, key := range keys {
		pattern := rules[key]
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compile rule %q: %w", key, err)
		}

		matches := re.FindStringSubmatch(body)
		if len(matches) == 0 {
			continue
		}
		if len(matches) > 1 {
			result[key] = matches[1]
			continue
		}
		result[key] = matches[0]
	}

	return result, nil
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
