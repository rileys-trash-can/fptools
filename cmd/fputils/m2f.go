package main

import (
	"github.com/rileys-trash-can/libfp"

	"github.com/go-audio/midi"
	"log"
	"math"
	"os"
)

func PlayMidi(printer *fp.Printer, file string) {
	f, err := os.Open(file)
	if err != nil {
		log.Fatalf("Fail open: %s", err)
	}

	d := midi.NewDecoder(f)

	err = d.Decode()
	if err != nil {
		log.Fatalf("Fail decode: %s", err)
	}

	var largest *midi.Track
	var length int

	log.Printf("Got %d tracks", len(d.Tracks))
	for i, e := range d.Tracks {
		log.Printf("  Track %2d has %d events and is %d", i, len(e.Events), e.Size)

		if len(e.Events) > length {
			length = len(e.Events)

			largest = e
		}
	}

	log.Printf("converting longest with %d events", len(largest.Events))

	track := largest

	const off = 0x8
	const on = 0x9
	var abstime uint32

	for i := 0; i < len(track.Events); i++ {
		e := track.Events[i]

		abstime += e.TimeDelta

		log.Printf("%5d %3d %4x dtime %5d  note %5d",
			abstime, i, e.MsgType, e.TimeDelta, e.Note,
		)

		// ignore empty?
		if e.Cmd == 0x2f {
			continue
		}

		var notelen uint32
		if e.MsgType == on {
			log.Print("turn on")

			note := e.Note

		inner:
			for ii := i; ii < len(track.Events); ii++ {
				notelen += track.Events[ii].TimeDelta

				if track.Events[ii].Note == note && track.Events[ii].MsgType == off {
					break inner
				}
			}

			log.Printf("turn off with dur %d", notelen)

			printer.Beep(fp.Sound{
				Freq: int(440 * math.Pow(2, (float64(note)-69)/12)),
				Dur:  int(notelen),
			})
		}

	}
}
