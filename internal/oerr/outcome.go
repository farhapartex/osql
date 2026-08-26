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
