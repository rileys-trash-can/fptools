package fp

import (
	"golang.org/x/image/bmp"

	"image"
	"image/color"

	"bytes"
	_ "embed"
	"log"
)

type subImage struct {
	img image.Image

	x, y             int
	offsetx, offsety int
}

func SubImage(i image.Image, w, h, offsetx, offsety int) image.Image {
	//log.Printf("w %4d h %4d offsetx %4d offsety %4d", w, h, offsetx, offsety)

	s := &subImage{i, w, h, offsetx, offsety}

	return s
}

func (s *subImage) ColorModel() color.Model {
	return s.img.ColorModel()
}

func (s *subImage) Bounds() image.Rectangle {
	b := s.img.Bounds()

	return image.Rectangle{
		Min: image.Pt(b.Min.X+s.offsetx, b.Min.Y+s.offsety),
		Max: image.Pt(b.Min.X+s.offsetx+s.x, b.Min.Y+s.offsety+s.y),
	}
}

const FIXOFFSET = -1

func (s *subImage) At(x, y int) color.Color {
	//tx, ty := x-s.offsetx, y-s.offsety
	tx, ty := s.x-(x-s.offsetx)+s.offsetx+FIXOFFSET,
		s.y-(y-s.offsety)+s.offsety+FIXOFFSET

	return s.img.At(tx, ty)
}

func (printer *Printer) PrintChunked(img image.Image, xoff, yoff int) (err error) {
	size := img.Bounds().Size()

	var totalx, totaly = size.X, size.Y

	var blocksizey = 100

	var blocksizex = totalx
	blocksizex = T(totalx < blocksizex, totalx, blocksizex)
	const DEBUGGAB = 0

	log.Printf("")
	log.Printf(" Printing using the following parameters:")
	log.Printf(" - width  %d", totalx)
	log.Printf(" - height %d", totaly)
	log.Printf("")
	log.Printf(" - blcksizex %d", blocksizex)
	log.Printf(" - blcksizey %d", blocksizey)

	for x := 0; x < totalx; x += blocksizex {
		for y := 0; y < totaly; y += blocksizey {
			// TODO: center or sth

			// prepare image
			err = printer.PrintPos(x+xoff, y+yoff)
			if err != nil {
				return
			}

			b := &bytes.Buffer{}
			i := SubImage(img,
				T((totalx-x) >= blocksizex, blocksizex, totalx-x)-DEBUGGAB,
				T((totaly-y) >= blocksizey, blocksizey, totaly-y)-DEBUGGAB,
				x, y,
			)

			err = bmp.Encode(b, i)
			if err != nil {
				return
			}

			log.Printf("PRBUF %d bytes || height %d", b.Len(), i.Bounds().Size().Y)
			err = printer.DirectPRBUF(b.Bytes())
			if err != nil {
				return
			}

			var res *Response
			res, err = printer.ReadResponse()
			if err != nil && res.Status != "Ok" {
				return
			}
		}
	}

	return nil
}
