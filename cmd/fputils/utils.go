package main

import (
	"github.com/rileys-trash-can/libfp"
	"github.com/rileys-trash-can/libfp/prbuf"

	// image stuffs
	_ "github.com/samuel/go-pcx/pcx"
	_ "golang.org/x/image/bmp"
	_ "image/jpeg"
	"image/png"

	"image"

	"bufio"
	_ "embed"
	"flag"
	"log"
	"os"
	"strings"
)

var (
	PrinterAddress = flag.String("host", os.Getenv("IPL_PRINTER"), "Specify printer, can also be set by envIPL_PRINTER ")

	OptBeep = flag.Bool("beep", true, "toggle connection-beep")

	OptDither     = flag.Bool("dither", true, "toggle dither when sending images")
	OptColorspace = flag.Bool("map-colorspace", true, "toggle colorspace conversion when sending images (only w/o dither)")

	OptResize = flag.String("resize", "fit", "set resize mode for images 'fit' or 'off'")
	OptPFC    = flag.Uint("count", 1, "amout of printfeeds / labels to print")
)

func main() {
	flag.Usage = func() {
		log.Print(Usage)
	}

	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		log.Fatalf(Usage)
	}

	switch args[0] {
	case "utf8encode":
		enc := fp.UTF8encode(strings.Join(args[1:], " "))

		os.Stdout.Write(enc)

	case "print":
		if len(args) < 2 {
			flag.Usage()
			os.Exit(1)
		}
		f, err := os.Open(args[1])
		if err != nil {
			log.Fatalf("Failed to open file %s: %s", args[1], err)
		}

		defer f.Close()
		log.Printf("opening printer")
		printer := OpenPrinter(args)

		// clear canvas
		err = printer.ClearCanvas(-1)
		if err != nil {
			return
		}

		log.Printf("sending")
		err = printer.Send(f)
		if err != nil {
			log.Printf("Failed to send file to printer: %s", err)
		}

		log.Printf("sent.")

		res, err := printer.Read()
		if err != nil {
			log.Printf("Failed to read response: %s", err)
		}

		log.Printf("res: %s", res)

	case "play":
		if len(args) < 2 {
			flag.Usage()
			os.Exit(1)
		}

		printer := OpenPrinter(args)
		PlayMidi(printer, args[1])

	case "textpipe":
		s := bufio.NewScanner(os.Stdin)
		p := OpenPrinter(args)

		for s.Scan() {
			p.ClearCanvas(-1)
			p.PrintPos(40, 0)
			p.PRText(strings.ReplaceAll(s.Text(), "\"", "\\\""))
			p.PF(*OptPFC)
		}

	case "encoderprbuf":
		if len(args) < 3 {
			flag.Usage()
			os.Exit(1)
		}

		infile, outfile := args[1], args[2]

		img := ReadImage(infile)

		// send image
		out, err := os.Create(outfile)
		if err != nil {
			log.Fatalf("Failed to write to %s: %s", outfile, err)
		}

		defer out.Close()

		prbuf.Encode(img, out)
		log.Printf("done.")

	case "decodeprbuf":
		if len(args) < 3 {
			flag.Usage()
			os.Exit(1)
		}

		infile, outfile := args[1], args[2]

		in, err := os.Open(infile)
		if err != nil {
			log.Fatalf("Failed to open: %s: %s", infile, err)
		}

		defer in.Close()

		img, err := prbuf.Decode(in)
		if err != nil {
			log.Fatalf("Failed to decode PRBUF: %s", err)
		}

		out, err := os.Create(outfile)
		if err != nil {
			log.Fatalf("Failed to out %s: %s", outfile, err)
		}

		err = png.Encode(out, img)
		if err != nil {
			log.Fatalf("Failed to encode out: %s", err)
		}

		defer out.Close()
		// send image

		log.Printf("Wrote a bunch of bytes")

	case "printimg":
		if len(args) < 2 {
			flag.Usage()
			os.Exit(1)
		}

		// 		conv := ipl.ImageConverter{
		// 			Dither:        *OptDither,
		// 			MapColorspace: *OptColorspace, // only works when dither is not set
		//
		// 			Resize: Resize(*OptResize),
		// 		}

		img := ReadImage(args[1])
		// b, err := conv.Convert(img) // fix sometime
		//if err != nil {
		//	log.Fatalf("Failed to convert image: %s", err)
		//}

		printer := OpenPrinter(args)

		// clear canvas
		err := printer.ClearCanvas(-1)
		if err != nil {
			return
		}

		/*
			w, h := img.Bounds().Dx(), img.Bounds().Dy()

			w = 835/2 - w/2
			h = 1412/2 - h/2

			log.Printf("w/h : %d/%d", w, h)
		*/
		const x, y = 0, 0
		// prepare image
		err = printer.PrintPos(x, y)
		if err != nil {
			log.Fatalf("Failed to set PrintPos: %s", err)
		}

		// send image
		err = printer.DirectImage(img)
		if err != nil {
			log.Fatalf("Failed to directimg: %s", err)
		}

		res, err := printer.ReadResponse()
		if err != nil && res.Status != "Ok" {
			log.Fatalf("Failed to readresponse: %s: %s", err, res.Status)
		}

		// play audio over http lul
		/*err = printer.SendCommand(`RUN "wget 'http://198.18.1.147:8080/pb.wav' -O /dev/dsp"`)
		if err != nil {
			return
		}

		res, err = printer.ReadResponse()
		if err != nil && res.Status != "Ok" {
			log.Fatalf("Failed to readresponse: %s: %s", err, res.Status)
		}
		*/

		log.Printf("sent data for printing...")
		err = printer.PF(*OptPFC)
		if err != nil {
			return
		}

		if err != nil && res.Status != "Ok" {
			log.Fatalf("Failed to readresponse: %s: %s", err, res.Status)
		}

	case "printprbuf":
		if len(args) < 2 {
			flag.Usage()
			os.Exit(1)
		}

		// 		conv := ipl.ImageConverter{
		// 			Dither:        *OptDither,
		// 			MapColorspace: *OptColorspace, // only works when dither is not set
		//
		// 			Resize: Resize(*OptResize),
		// 		}

		in, err := os.ReadFile(args[1])
		if err != nil {
			log.Fatalf("Failed to open %s: %s", args[1], err)
		}

		printer := OpenPrinter(args)

		// clear canvas
		err = printer.ClearCanvas(-1)
		if err != nil {
			return
		}

		// prepare image
		err = printer.SendCommand("PP 0,0:II:MAG 1,1")
		if err != nil {
			return
		}

		res, err := printer.ReadResponse()
		if err != nil && res.Status != "Ok" {
			log.Fatalf("Failed to readresponse: %s: %s", err, res.Status)
		}

		// send imagesize
		err = printer.DirectPRBUF(in)
		if err != nil {
			log.Fatalf("Failed to directPRBUF: %s", err)
		}

		res, err = printer.ReadResponse()
		if err != nil && res.Status != "Ok" {
			log.Fatalf("Failed to readresponse: %s: %s", err, res.Status)
		}

		log.Printf("sent data for printing...")
		err = printer.PF(*OptPFC)
		if err != nil {
			log.Fatalf("Err: %s", err)
		}

	case "sendimg":
		if len(args) < 3 {
			flag.Usage()
			os.Exit(1)
		}

		conv := fp.ImageConverter{
			Dither:        *OptDither,
			MapColorspace: *OptColorspace, // only works when dither is not set

			Resize: Resize(*OptResize),
		}

		img := ReadImage(args[2])
		b, err := conv.Convert(img)
		if err != nil {
			log.Fatalf("Failed to convert image: %s", err)
		}

		printer := OpenPrinter(args)
		err = printer.LoadImageByte(args[1], b)
		if err != nil {
			log.Fatalf("Failed to send img: %s", err)
		}

		log.Printf("sent.")

		res, err := printer.Read()
		if err != nil {
			log.Printf("Failed to read response: %s", err)
		}

		log.Printf("res: %s", res)

	case "help":
		log.Printf(Usage)
	default:
		log.Fatalf(Usage)
	}
}

//go:embed help.txt
var Usage string

func ReadImage(name string) (i image.Image) {
	f, err := os.Open(name)
	if err != nil {
		log.Fatalf("Failed to read file '%s': %s", name, err)
	}

	defer f.Close()

	i, fm, err := image.Decode(f)
	if err != nil {
		log.Fatalf("Failed to read file '%s': %s", name, err)
	}

	log.Printf("Decoded image in %s format", fm)

	return
}

func Resize(r string) fp.Resize {
	switch r {
	case "off":
		return fp.ResizeOff
	case "fit":
		return fp.ResizeFit

	default:
		log.Fatalf("--resize has to be one of 'off' and 'fit'")
	}

	return 0
}
