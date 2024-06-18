package main

import (
	"flag"
	"net/http"

	"bytes"
	_ "embed"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rileys-trash-can/libfp"
	"io"
	"log"
	"os"
	"time"

	// image stuffs
	_ "github.com/samuel/go-pcx/pcx"
	_ "golang.org/x/image/bmp"
	"image"
	_ "image/jpeg"
	_ "image/png"
)

var (
	//go:embed index.txt
	eIndex []byte
)

var (
	ListenAddr = flag.String("listen", "[::]:8070", "specify port to listen on")

	PrinterAddressHost = flag.String("host", os.Getenv("IPL_PRINTER"), "Specify printer, can also be set by env IPL_PRINTER (net port)")
	PrinterAddressPort = flag.String("port", os.Getenv("IPL_PORT"), "Specify printer, can also be set by env IPL_PORT (usb port)")

	PrinterAddressType = flag.String("ctype", os.Getenv("IPL_CTYPE"), "Specify printer connection type, can also be set by env IPL_CTYPE")

	OptBeep = flag.Bool("beep", true, "toggle connection-beep")
)

var printer *fp.Printer

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	flag.Parse()

	printer = OpenPrinter()

	gmux := mux.NewRouter()

	gmux.Path("/").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleIndex)))

	gmux.Path("/print").
		Methods("PUT").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handlePrint)))

	log.Fatalf("Failed to ListenAndServe: %s",
		http.ListenAndServe(*ListenAddr, gmux))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/plain")

	w.Write(eIndex)
}

func handlePrint(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	b := bytes.NewBuffer(data)

	img, fmt, err := image.Decode(b)
	if err != nil {
		panic(err)
	}

	log.Printf("Received Image in %s format bounds: %+v", fmt, img.Bounds())

	err = os.WriteFile(time.Now().Format(time.RFC3339)+"."+fmt, data, 0755)
	if err != nil {
		panic(err)
	}

	err = printer.PrintChunked(img, 0, 0)
	if err != nil {
		panic(err)
	}

	err = printer.PF(1)
	if err != nil {
		panic(err)
	}
}

var ErrorHandlerMiddleware = mux.MiddlewareFunc(func(next http.Handler) http.Handler {
	return &errMiddleware{next}
})

type errMiddleware struct {
	next http.Handler
}

func (m *errMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := recover()

		log.Printf("Error in request of '%s': %s", r.URL.Path, err)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(500)

		fmt.Fprintf(w, "There was an error handeling your request: %s\n", err)
	}()

	m.next.ServeHTTP(w, r)
}
