package fp

import (
	"bytes"
	"fmt"
)

type Sound struct {
	Freq int // in Hz
	Dur  int // in ms, step is 20ms
}

func (p *Printer) Beep(notes ...Sound) (err error) {
	buf := new(bytes.Buffer)

	for i := 0; i < len(notes); i++ {
		fmt.Fprintf(buf, "SOUND %d,%d ",
			notes[i].Freq, notes[i].Dur/20,
		)

		if i != len(notes)-1 {
			fmt.Fprintf(buf, ": ")
		}
	}

	err = p.SendCommand(buf.String())
	if err != nil {
		return
	}

	_, err = p.ReadResponse()
	return
}

func T[K any](c bool, a, b K) K {
	if c {
		return a
	} else {
		return b
	}
}
