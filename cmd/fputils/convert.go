package main

import (
	"image"
)

func fImage(in image.Image) image.Image {
	out := image.NewGray(image.Rect(0, 0, 840, 1260))

	return out
}
