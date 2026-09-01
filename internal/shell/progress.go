package shell

import (
	"fmt"
	"io"
)

const eraseLine = "\r\033[K"

type scanProgress struct {
	out     io.Writer
	scanned int
	shown   bool
}

func (p *scanProgress) report(scanned int) {
	p.scanned = scanned
	if p.out == nil {
		return
	}
	fmt.Fprintf(p.out, "\rscanned %d…", scanned)
	p.shown = true
}

func (p *scanProgress) erase() {
	if !p.shown {
		return
	}
	fmt.Fprint(p.out, eraseLine)
	p.shown = false
}
