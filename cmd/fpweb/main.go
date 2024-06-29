package main

import (
	"flag"
	"net/http"

	"bytes"
	"encoding/base64"
	"github.com/gorilla/mux"
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/rileys-trash-can/libfp"
	"log"

	// image stuffs
	_ "github.com/samuel/go-pcx/pcx"
	_ "golang.org/x/image/bmp"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
)

var printer *fp.Printer

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	flag.Parse()
	conf := GetConfig()

	// verify DB is valid
	GetDB()

	if !*OptDryRun {
		printer = OpenPrinter()
	}

	gmux := mux.NewRouter()

	// static stuff
	gmux.Path("/").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(&handleFile{"text/html", eIndex}))

	gmux.Path("/bs.css").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(&handleFile{"text/css", eBScss}))

	gmux.Path("/api").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(&handleFile{"text/plain", eIndexApi}))

	// ui stuff
	gmux.Path("/img/{uuid}").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleGetImg)))

	gmux.Path("/job/{uuid}").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleJob)))

	gmux.Path("/api/print").
		Methods("POST").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handlePrintPOST)))

	// api stuff
	gmux.Path("/api/print").
		Methods("PUT").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handlePrint)))

	gmux.Path("/api/job/{uuid}").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleJobAPI)))

	gmux.Path("/api/list").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleList)))

	addr := T(*ListenAddr != "", *ListenAddr, conf.Listen)

	if addr == "" {
		addr = "[::]:8070"
	}

	log.Printf("Listening on %s", addr)
	log.Fatalf("Failed to ListenAndServe: %s",
		http.ListenAndServe(addr, gmux))
}

type Filter interface {
	Apply(img image.Image) image.Image
}

type PixMapperFilter struct {
	mapper *dither.Ditherer
}

func (mf *PixMapperFilter) Apply(src image.Image) image.Image {
	return mf.mapper.Dither(src)
}

func DitherFromString(n string) Filter {
	switch n {
	case "o4x4": // to dither
		mapper := dither.NewDitherer([]color.Color{color.White, color.Black})
		mapper.Mapper = dither.PixelMapperFromMatrix(dither.ClusteredDot4x4, 1.0)

		return &PixMapperFilter{mapper: mapper}

	case "noise": // to dither
		mapper := dither.NewDitherer([]color.Color{color.White, color.Black})
		mapper.Mapper = dither.RandomNoiseGrayscale(.1, .5)

		return &PixMapperFilter{mapper: mapper}

	case "bayer": // to dither
		mapper := dither.NewDitherer([]color.Color{color.White, color.Black})
		mapper.Mapper = dither.Bayer(3, 3, .6)

		return &PixMapperFilter{mapper: mapper}
	}

	return nil
}

func BoolFromString(n string) bool {
	switch n {
	case "on":
		return true
	case "true":
		return true
	}

	return false
}

func b64image(img image.Image) string {
	b := &bytes.Buffer{}
	err := png.Encode(b, img)
	if err != nil {
		panic(err)
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

type ImageList struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
	Total  int `json:"total"`

	Images []Image `json:"images"`
}

func T[K any](c bool, a, b K) K {
	if c {
		return a
	}

	return b
}
