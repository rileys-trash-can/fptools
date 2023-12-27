package main

import (
	"github.com/rileys-trash-can/libfp"

	_ "embed"
	"log"
	"os"
)

func OpenPrinter(args []string) *fp.Printer {
	host := *PrinterAddress
	if host == "" {
		var ok bool
		host, ok = os.LookupEnv("FP_PRINTER")
		if !ok {
			log.Fatalf("no printer specified: both env->FP_PRINTER and --host was not specified")
		}
	}

	log.Printf("Dialing %s", host)
	p, err := fp.DialPrinter(host)
	if err != nil {
		log.Fatalf("Printer %s", err)
	}

	if *OptBeep {
		err := p.Beep(fp.Sound{Freq: 850, Dur: 200}, fp.Sound{Freq: 950, Dur: 200})
		if err != nil {
			log.Fatalf("Failed to communicate with printer: Beep: %s", err)
		}
	}

	return p
}
