package output

import "github.com/farhapartex/osql/internal/textwidth"

const ellipsis = textwidth.Ellipsis

func DisplayWidth(s string) int { return textwidth.Of(s) }

func RuneWidth(r rune) int { return textwidth.Rune(r) }

func padRight(s string, width int) string { return textwidth.PadRight(s, width) }

func padLeft(s string, width int) string { return textwidth.PadLeft(s, width) }

func truncateMiddle(s string, width int) string { return textwidth.TruncateMiddle(s, width) }
