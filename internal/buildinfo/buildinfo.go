package buildinfo

import "fmt"

func String(version, commit string) string {
	if version == "" {
		version = "dev"
	}
	if commit == "" {
		commit = "none"
	}
	return fmt.Sprintf("osql %s (%s)", version, commit)
}
