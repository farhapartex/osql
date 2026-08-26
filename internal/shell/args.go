package shell

import "strings"

func SplitArgs(line string) []string {
	args := []string{}
	var current strings.Builder
	quote := byte(0)
	started := false

	for i := 0; i < len(line); i++ {
		c := line[i]

		switch {
		case quote == 0 && (c == ' ' || c == '\t'):
			if started {
				args = append(args, current.String())
				current.Reset()
				started = false
			}
		case quote == 0 && (c == '\'' || c == '"'):
			quote = c
			started = true
		case quote != 0 && c == quote:
			quote = 0
		case c == '\\' && quote != '\'' && i+1 < len(line):
			i++
			current.WriteByte(line[i])
			started = true
		default:
			current.WriteByte(c)
			started = true
		}
	}

	if started {
		args = append(args, current.String())
	}
	return args
}
