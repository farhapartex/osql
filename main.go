package main

import (
	"fmt"

	"github.com/farhapartex/osql/internal/buildinfo"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	fmt.Println(buildinfo.String(version, commit))
}
