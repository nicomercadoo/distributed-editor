package repl

import (
	"errors"
	"strings"
)

type Command struct {
	Name string
	Args []string
	Raw  string
}

func Parse(line string) (Command, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return Command{}, errors.New("empty")
	}

	fields := splitFields(trimmed)
	if len(fields) == 0 {
		return Command{}, errors.New("empty")
	}

	return Command{
		Name: strings.ToLower(fields[0]),
		Args: fields[1:],
		Raw:  trimmed,
	}, nil
}

func splitFields(input string) []string {
	var out []string
	var b strings.Builder
	var quote rune
	var escaped bool

	for _, r := range input {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			b.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t':
			if b.Len() > 0 {
				out = append(out, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}

	if b.Len() > 0 {
		out = append(out, b.String())
	}

	return out
}
