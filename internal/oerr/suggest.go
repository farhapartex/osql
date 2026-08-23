package oerr

const maxSuggestionDistance = 2

func Distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) < len(rb) {
		ra, rb = rb, ra
	}
	if len(rb) == 0 {
		return len(ra)
	}

	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}

	return prev[len(rb)]
}

func Suggest(got string, candidates []string) (string, bool) {
	best := ""
	bestDistance := maxSuggestionDistance + 1

	for _, c := range candidates {
		d := Distance(got, c)
		if d > maxSuggestionDistance {
			continue
		}
		if d < bestDistance || (d == bestDistance && c < best) {
			best, bestDistance = c, d
		}
	}

	return best, best != ""
}
