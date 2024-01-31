package fp

import (
	"fmt"
	"strings"
)

func (p *Printer) PrintPos(x, y int) (err error) {
	err = p.SendCommand(fmt.Sprintf("PP %d,%d", x, y))
	if err != nil {
		return
	}

	_, err = p.ReadResponse()

	return
}

func (p *Printer) PRText(txt string) (err error) {
	err = p.SendCommand(fmt.Sprintf("PRTXT \"%s\"",
		strings.ReplaceAll(strings.ReplaceAll(txt, "\n", "\\n"), "\"", "\\\"")))
	if err != nil {
		return
	}

	_, err = p.ReadResponse()
	return
}
