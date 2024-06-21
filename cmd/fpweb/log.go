package main

import (
	_ "embed"
	"github.com/rileys-trash-can/libfp/prbuf"
	"io"
	"log"
	"os"
	// image stuffs
	"errors"
	"github.com/samuel/go-pcx/pcx"
	"golang.org/x/image/bmp"
	"image"
	"image/jpeg"
	"image/png"
)

func SaveImage(img image.Image, name string) (string, error) {
	p := GetConfig().Saves + name + ".png"

	return p, saveImage(p, img, "png")
}

func saveImage(name string, img image.Image, format string) error {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return err
	}

	defer f.Close()

	return encodeImage(f, img, format)
}

func encodeImage(w io.Writer, img image.Image, fmt string) (err error) {
	log.Printf("encoding Image")

	switch fmt {
	case "png":
		return png.Encode(w, img)

	case "jpg":
	case "jpeg":
		return jpeg.Encode(w, img, &jpeg.Options{
			Quality: 50,
		})

	case "pcx":
		return pcx.Encode(w, img)

	case "bmp":
		return bmp.Encode(w, img)

	case "prbuf":
		prbuf.Encode(img, w)
		return nil
	}

	return errors.New("Unknown Image Format")
}
