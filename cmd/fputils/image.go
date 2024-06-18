package main

import (
	"image"
	"image/color"
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
