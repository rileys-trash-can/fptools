package main

import (
	"flag"
	"net/http"

	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/rileys-trash-can/libfp"
	"io"
	"log"
	"os"
	"runtime/debug"
	"text/template"

	// image stuffs
	_ "github.com/samuel/go-pcx/pcx"
	_ "golang.org/x/image/bmp"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
)

var (
	//go:embed index.txt
	eIndexApi []byte

	//go:embed bs.css
	eBScss []byte

	//go:embed index.html
	eIndex []byte

	//go:embed index.tmpl.html
	eStatus string

	tStatus *template.Template = func() *template.Template {
		templ, err := template.New("status").Parse(eStatus)
		if err != nil {
			panic(err)
		}

		return templ
	}()
)

var (
	ListenAddr = flag.String("listen", "[::]:8070", "specify port to listen on")

	PrinterAddressHost = flag.String("host", os.Getenv("IPL_PRINTER"), "Specify printer, can also be set by env IPL_PRINTER (net port)")
	PrinterAddressPort = flag.String("port", os.Getenv("IPL_PORT"), "Specify printer, can also be set by env IPL_PORT (usb port)")

	PrinterAddressType = flag.String("ctype", os.Getenv("IPL_CTYPE"), "Specify printer connection type, can also be set by env IPL_CTYPE")

	OptBeep   = flag.Bool("beep", true, "toggle connection-beep")
	OptDryRun = flag.Bool("dry-run", false, "disables connection to printer; for testing")
)

var printer *fp.Printer

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	flag.Parse()

	if !*OptDryRun {
		printer = OpenPrinter()
	}

	gmux := mux.NewRouter()

	gmux.Path("/api").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(&handleFile{"text/plain", eIndexApi}))

	gmux.Path("/").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(&handleFile{"text/html", eIndex}))

	gmux.Path("/bs.css").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(&handleFile{"text/css", eBScss}))

	gmux.Path("/api/print").
		Methods("PUT").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handlePrint)))

	gmux.Path("/api/status/{uuid}").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleStatusAPI)))

	gmux.Path("/img/{uuid}").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleGetImg)))

	gmux.Path("/status/{uuid}").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleStatus)))

	gmux.Path("/api/print").
		Methods("POST").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handlePrintPOST)))

	log.Printf("Listening on %s", *ListenAddr)
	log.Fatalf("Failed to ListenAndServe: %s",
		http.ListenAndServe(*ListenAddr, gmux))
}

type handleFile struct {
	contenttype string
	data        []byte
}

func (hf *handleFile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", hf.contenttype)
	w.WriteHeader(200)

	w.Write(hf.data)

	return
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

	uuid := uuid.New()

	path, err := SaveImage(img, uuid.String())
	if err != nil {
		panic(err)
	}

	log.Printf("saved image to %s", path)

	newImageCh <- uuid

	if !*OptDryRun {
		err = printer.PrintChunked(img, 0, 0)
		if err != nil {
			panic(err)
		}

		err = printer.PF(1)
		if err != nil {
			panic(err)
		}
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
		if err == nil {
			return
		}

		log.Printf("Error in %s request of '%s': %s", r.Method, r.URL.Path, err)
		log.Print("stacktrace from panic: \n" + string(debug.Stack()))

		switch w.Header().Get("Content-Type") {
		case "application/json":
			w.Header().Set("Location", "/")

			w.WriteHeader(500)
			fmt.Fprintf(w, "{\"done\":false,\"message\":\"%s\",\"error\":true}", err)
			return

		default:
			w.Header().Set("Content-Type", "text/plain")
			fallthrough

		case "text/plain":
			w.WriteHeader(500)
			fmt.Fprintf(w, "There was an error handeling your request: %s\n return to / to do stuff", err)
		}

	}()

	m.next.ServeHTTP(w, r)
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
	if n == "on" {
		return true
	}

	return false
}

type StatusResponse struct {
	Message string `json:"message"`
	Done    bool   `json:"done"`
	Reload  bool   `json:"reload"`
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "text/html")

	id, ok := vars["uuid"]
	if !ok {
		panic("no id specified")
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		panic(err)
	}

	status := GetStatus(uid)
	if status == nil {
		panic("Invalid Status // uid unknown")
	}

	err = tStatus.Execute(w, StatusPageArgs{
		PrintID:       uid.String(),
		InitialStatus: status.String(),
	})
}

func handleStatusAPI(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	id, ok := vars["uuid"]
	if !ok {
		panic("no id specified")
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		panic(err)
	}

	status := GetStatus(uid)
	if status == nil {
		panic("Invalid Status")
	}

	w.WriteHeader(200)

	enc := json.NewEncoder(w)
	err = enc.Encode(&StatusResponse{
		Message: status.String(),
		Done:    status.Done,
		Reload:  status.Reload,
	})
	if err != nil {
		panic(err)
	}
}

func handlePrintPOST(w http.ResponseWriter, r *http.Request) {
	uid := uuid.New()

	newImageCh <- uid
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)

	fmt.Fprintf(w, `<head>
	  <meta http-equiv="Refresh" content="0; URL=/status/%s" />
	</head>`, uid)

	file, header, err := r.FormFile("file")
	if err != nil {
		panic(err)
	}

	go func() {
		const totalSteps = 8

		defer file.Close()

		log.Printf("[POST] file '%s' %d bytes", header.Filename, header.Size)

		var (
			ditherer = DitherFromString(r.FormValue("dither"))

			optresize  = BoolFromString(r.FormValue("resize"))
			optstretch = BoolFromString(r.FormValue("stretch"))
			optrotate  = BoolFromString(r.FormValue("rotate"))
			optcenterh = BoolFromString(r.FormValue("centerh"))
			optcenterv = BoolFromString(r.FormValue("centerv"))
		//	opttiling  = BoolFromString(r.FormValue("tiling"))
		)

		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "decode",
			Progress: 1 / totalSteps,
			Done:     false,
		}
		img, ifmt, err := image.Decode(file)
		if err != nil {
			panic(err)
		}

		log.Printf("[POST] Received Image in %s format bounds: %+v", ifmt, img.Bounds())

		//TODO more config
		var maxwidth, maxheight = 800, 1200
		var method = imaging.Lanczos

		size := img.Bounds().Size()

		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "rotating",
			Progress: 2 / totalSteps,
			Done:     false,
		}
		if optrotate {
			log.Printf("testing rotate")
			if (maxwidth > maxheight) != (size.X > size.Y) {
				log.Printf("rotating...")
				img = imaging.Rotate90(img)
			}
		}

		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "resizing",
			Progress: 3 / totalSteps,
			Done:     false,
		}
		if optresize {
			log.Printf("resize; stretch: %t", optstretch)
			if optstretch {
				img = imaging.Resize(img, maxwidth, maxheight, method)
			} else {
				size = img.Bounds().Size()

				px := float32(size.X) / float32(maxwidth)
				py := float32(size.Y) / float32(maxheight)

				if px > py {
					img = imaging.Resize(img, maxwidth, 0, method)
				} else {
					img = imaging.Resize(img, 0, maxheight, method)
				}
			}
		}

		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "centering",
			Progress: 4 / totalSteps,
			Done:     false,
		}
		if optcenterh || optcenterv {
			nimg := imaging.New(maxwidth, maxheight, color.White)
			size = img.Bounds().Size()

			var x, y = 0, 0
			if optcenterh {
				x = maxwidth/2 - size.X/2
			}

			if optcenterv {
				y = maxheight/2 - size.Y/2
			}

			draw.Draw(nimg,
				img.Bounds().Add(image.Pt(x, y)),
				img,
				image.Point{},
				draw.Over,
			)

			img = nimg
		}

		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "dithering",
			Progress: 5 / totalSteps,
			Done:     false,
		}
		if ditherer != nil {
			log.Printf("Dithering with %T", ditherer)

			img = ditherer.Apply(img)
		}

		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "saving",
			Progress: 6 / totalSteps,
			Done:     false,
		}
		_, err = SaveImage(img, uid.String())
		if err != nil {
			log.Printf("Failed to encode & save image: %s", err)
		}

		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "printing",
			Progress: 7 / totalSteps,
			Reload:   true,
			Done:     false,
		}

		if !*OptDryRun {
			err = printer.PrintChunked(img, 0, 0)
			if err != nil {
				panic(err)
			}

			err = printer.PF(1)
			if err != nil {
				panic(err)
			}
		}

		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "done",
			Progress: 8 / totalSteps,
			Reload:   true,
			Done:     true,
		}
	}()
}

func b64image(img image.Image) string {
	b := &bytes.Buffer{}
	err := png.Encode(b, img)
	if err != nil {
		panic(err)
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

type StatusPageArgs struct {
	InitialStatus string
	PrintID       string
}

func handleGetImg(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")

	v := mux.Vars(r)
	t, ok := v["uuid"]
	if !ok {
		panic("no UUID")
	}

	uid, err := uuid.Parse(t)
	if err != nil {
		panic(err)
	}

	log.Printf("Serving %s", "saves/"+uid.String()+".png")
	http.ServeFile(w, r, "saves/"+uid.String()+".png")
}
