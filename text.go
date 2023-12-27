package fp

import (
	"fmt"
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
	err = p.SendCommand(fmt.Sprintf("PRTXT \"%s\"", txt))
	if err != nil {
		return
	}

	_, err = p.ReadResponse()
	return
}
