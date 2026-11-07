package motion

import (
	"fmt"
	"regexp"
)

// latinExtensions are the characters whose Script_Extensions has Latin although
// their Script does not.
const latinExtensions = `\x{b7}\x{2bc}\x{2c7}\x{2c9}-\x{2cb}\x{2cd}\x{2d7}\x{2d9}\x{300}-\x{30e}\x{310}\x{311}` +
	`\x{313}\x{320}\x{323}-\x{325}\x{32d}\x{32e}\x{330}\x{331}\x{358}\x{35e}\x{363}-\x{36f}\x{485}\x{486}` +
	`\x{951}\x{952}\x{10fb}\x{1df8}\x{202f}\x{20f0}\x{2e17}\x{a700}-\x{a707}\x{a92e}`

var scriptProperty = regexp.MustCompile(`\\([pP])\{(?:Script_Extensions|scx|Script|sc)=(\w+)\}`)

// compile reads the regular expressions of the g:fuzzy_motion_*_regexp_list
// options. Go has no Script_Extensions, so those properties become classes.
func compile(pattern string) (*regexp.Regexp, error) {
	translated := scriptProperty.ReplaceAllStringFunc(pattern, func(s string) string {
		m := scriptProperty.FindStringSubmatch(s)
		switch {
		case m[2] != "Latin":
			return `\` + m[1] + `{` + m[2] + `}`
		case m[1] == "p":
			return `[\p{Latin}` + latinExtensions + `]`
		}
		return `[^\p{Latin}` + latinExtensions + `]`
	})
	re, err := regexp.Compile(translated)
	if err != nil {
		return nil, fmt.Errorf("fuzzy-motion: %w", err)
	}
	return re, nil
}
