package oerr

import "fmt"

func NoMatches() string {
	return "No files matched."
}

func EmptyFolder(path string) string {
	return fmt.Sprintf("'%s' is empty.", path)
}

func NoApps() string {
	return "I didn't find any installed apps."
}

func NoAppsMatched() string {
	return "No apps matched."
}

func LimitReached(limit int) string {
	return fmt.Sprintf("Showing the first %d. Raise the limit to see more.", limit)
}

func SortTruncated(kept int) string {
	return fmt.Sprintf("Sorted the first %d matches. Add a limit to be sure you are seeing the top.", kept)
}
