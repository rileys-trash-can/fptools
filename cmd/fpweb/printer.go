package main

import (
	"github.com/rileys-trash-can/libfp"

	_ "embed"
	"log"
)

func OpenPrinter() *fp.Printer {
	host := *PrinterAddressHost
	port := *PrinterAddressPort

	ctype := *PrinterAddressType

	var err error
	var p *fp.Printer

	switch ctype {
	case "net":
		log.Printf("Dialing %s", host)
		p, err = fp.DialPrinter(host)
		if err != nil {
			log.Fatalf("Printer %s", err)
		}

	case "serial":
		log.Printf("Open %s", port)
		p, err = fp.OpenPrinter(port)
		if err != nil {
			log.Fatalf("Printer %s", err)
		}

	default:
		log.Fatalf("Invaid connection type '%s', choose between 'net' and 'serial'", ctype)
	}

	if *OptBeep {
		err := p.Beep(fp.Sound{Freq: 850, Dur: 200}, fp.Sound{Freq: 950, Dur: 200})
		if err != nil {
			log.Fatalf("Failed to communicate with printer: Beep: %s", err)
		}
	}

	return p
}
