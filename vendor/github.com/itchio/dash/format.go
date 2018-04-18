package dash

import (
	"encoding/json"
	"fmt"
	"strings"

	humanize "github.com/dustin/go-humanize"
)

func (v *Verdict) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf(`| %s %s`,
		humanize.IBytes(uint64(v.TotalSize)),
		v.BasePath))

	for _, candidate := range v.Candidates {
		lines = append(lines, candidate.String())
	}

	return strings.Join(lines, "\n")
}

func (c *Candidate) String() string {
	line := fmt.Sprintf(`|-- %s %s %s-%s`,
		humanize.IBytes(uint64(c.Size)),
		c.Path,
		c.Flavor,
		c.Arch,
	)

	append := func(label string, input interface{}) {
		var values []string

		if input == nil {
			return
		}

		switch node := input.(type) {
		case map[string]interface{}:
			for k, v := range node {
				if b, ok := v.(bool); ok {
					if b {
						values = append(values, k)
					}
				} else {
					values = append(values, fmt.Sprintf("%s=%s", k, v))
				}
			}
		default:
			return
		}

		var valueString = ""
		if len(values) > 0 {
			valueString = fmt.Sprintf("(%s)", strings.Join(values, " "))
		}
		line += fmt.Sprintf(" %s%s", label, valueString)
	}

	marshalled, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}

	intermediate := make(map[string]interface{})
	err = json.Unmarshal(marshalled, &intermediate)
	if err != nil {
		return err.Error()
	}

	append("win", intermediate["windowsInfo"])
	append("sh", intermediate["scriptInfo"])

	return line
}
