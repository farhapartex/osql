package output

import "strings"

var wideRanges = [][2]rune{
	{0x1100, 0x115F},
	{0x2E80, 0x303E},
	{0x3041, 0x33FF},
	{0x3400, 0x4DBF},
	{0x4E00, 0x9FFF},
	{0xA000, 0xA4CF},
	{0xAC00, 0xD7A3},
	{0xF900, 0xFAFF},
	{0xFE30, 0xFE6F},
	{0xFF00, 0xFF60},
	{0xFFE0, 0xFFE6},
	{0x1F300, 0x1F64F},
	{0x1F900, 0x1F9FF},
	{0x20000, 0x3FFFD},
}

func RuneWidth(r rune) int {
	for _, span := range wideRanges {
		if r >= span[0] && r <= span[1] {
			return 2
		}
	}
	return 1
}

func DisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		width += RuneWidth(r)
	}
	return width
}

func padRight(s string, width int) string {
	if gap := width - DisplayWidth(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

func padLeft(s string, width int) string {
	if gap := width - DisplayWidth(s); gap > 0 {
		return strings.Repeat(" ", gap) + s
	}
	return s
}

func truncateMiddle(s string, width int) string {
	if DisplayWidth(s) <= width {
		return s
	}

	keep := width - DisplayWidth(ellipsis)
	if keep < 2 {
		return ellipsis
	}

	headBudget := (keep + 1) / 2
	tailBudget := keep - headBudget

	runes := []rune(s)
	head := 0
	used := 0
	for head < len(runes) && used+RuneWidth(runes[head]) <= headBudget {
		used += RuneWidth(runes[head])
		head++
	}

	tail := len(runes)
	used = 0
	for tail > head && used+RuneWidth(runes[tail-1]) <= tailBudget {
		used += RuneWidth(runes[tail-1])
		tail--
	}

	return string(runes[:head]) + ellipsis + string(runes[tail:])
}
