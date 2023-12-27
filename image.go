package fp

import (
	"bytes"
	"github.com/rileys-trash-can/libfp/prbuf"

	// image stuffs
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/nfnt/resize"
	"github.com/samuel/go-pcx/pcx"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	_ "embed"
	"fmt"
	"log"
)

// PRBUF<nexp1>[,<nexp2 ]<new line><image data>
func (p *Printer) DirectPRBUF(d []byte) (err error) {
	err = p.SendCommand(fmt.Sprintf("PRBUF %d", len(d)))
	if err != nil {
		return
	}

	p.WriteAll(d)

	return
}

// borked, uploads PCX data, printer no accept as image :(
func (p *Printer) LoadImage(name string, i image.Image) (err error) {
	d, err := DefaultConverter.Convert(i)
	if err != nil {
		return
	}

	return p.LoadImageByte(name, d)
}

// only supports PCX images
// use LoadImage to import *image.Image s
func (p *Printer) LoadImageByte(name string, d []byte) (err error) {
	err = p.SendCommand(fmt.Sprintf("IMAGE LOAD 0,\"%s\",%d,\"\"", name, len(d)))
	if err != nil {
		return
	}

	err = p.WriteAll(d)
	if err != nil {
		return
	}
	log.Printf("wrote %d", len(d))

	err = p.SendCommand("")
	if err != nil {
		return
	}

	res, e := p.ReadResponse()
	if e != nil {
		log.Printf("err: %s", e)
	}

	log.Printf("%v", res)

	err = p.SendCommand("FILES")
	if err != nil {
		return
	}

	res, e = p.ReadResponse()
	if e != nil {
		log.Printf("err: %s", e)
	}

	log.Printf("%v", res)

	return
}

type Resize uint8

const (
	ResizeOff Resize = iota
	ResizeFit
)

type ImageConverter struct {
	Dither        bool
	MapColorspace bool // only works when dither is not set

	Resize Resize
}

var DefaultConverter = &ImageConverter{
	Dither: true,

	Resize: ResizeOff,
}

// converts image i to a PCX encoded image
func (conv *ImageConverter) Convert(i image.Image) (b []byte, err error) {
	// try to cast image to *image.RGBA
	rgba, ok := i.(*image.RGBA)
	if !ok || rgba == nil { // convert to *image.RGBA
		// fill blank with white
		rgba = image.NewRGBA(i.Bounds())

		// draw image
		draw.Draw(rgba, rgba.Bounds(), i, image.Point{}, draw.Over)
	}

	// fill background with white
	draw.DrawMask(rgba,
		rgba.Bounds(),
		rgba,
		image.Point{},
		&image.Uniform{color.Transparent},
		image.Point{},
		draw.Over)

	w, h := uint(rgba.Bounds().Dx()), uint(rgba.Bounds().Dy())
	// Calculate the scaling factors for width and height

	var scale float64

	const (
		maxWidth  = 807
		maxHeight = 1214
	)

	if w > maxWidth {
		scale = float64(807) / float64(w)

		w = uint(float64(w) * scale)
		h = uint(float64(h) * scale)
	} else if h > maxHeight {
		scale = float64(maxHeight) / float64(h)

		w = uint(float64(w) * scale)
		h = uint(float64(h) * scale)
	} else {
		w = uint(float64(w) * scale)
		h = uint(float64(h) * scale)
	}

	log.Printf("resizing")
	img := resize.Resize(w, h, rgba, resize.Bicubic)

	// dither B/W:
	if conv.Dither {
		dit := dither.NewDitherer([]color.Color{color.White, color.Black})
		dit.Mapper = dither.Bayer(8, 8, 1.0)
		img = dit.Dither(img)
	} else if conv.MapColorspace {

		// B&W ify
		for y := img.Bounds().Min.Y; y < rgba.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < rgba.Bounds().Max.X; x++ {
				// Get the color of the pixel in the original image
				originalColor := rgba.At(x, y)

				// Convert the color to the desired color space
				newColor := prbuf.BWModel.Convert(originalColor)

				// Set the color of the corresponding pixel in the new image
				rgba.Set(x, y, newColor)
			}
		}
	}

	// encode pcx
	pcxb := new(bytes.Buffer)

	err = pcx.Encode(pcxb, rgba)
	if err != nil {
		return
	}

	//debugging
	//os.WriteFile("out.pcx", pcxb.Bytes(), 0755)

	return pcxb.Bytes(), err
}

// PRBUF<nexp1>[,<nexp2 ]<new line><image data>
func (p *Printer) DirectImage(i image.Image) (err error) {
	buf := &bytes.Buffer{}

	prbuf.Encode(i, buf)
	d := buf.Bytes()

	log.Printf("%d bytes of prbuf", len(d))

	return p.DirectPRBUF(d)
}
