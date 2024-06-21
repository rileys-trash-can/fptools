package main

import (
	"flag"
	"net/http"

	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/rileys-trash-can/libfp"
	"io"
	"log"
	"os"
	"runtime/debug"
	"strconv"
	"text/template"

	// image stuffs
	_ "github.com/samuel/go-pcx/pcx"
	_ "golang.org/x/image/bmp"
	"image"
	"image/color"
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

	//go:embed jobstatus.template.html
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
	ConfigPath = flag.String("config", "config.yml", "config to read")

	ListenAddr = flag.String("listen", "", "specify port to listen on, fallback is [::]:8070")

	PrinterAddressHost = flag.String("host", os.Getenv("IPL_PRINTER"), "Specify printer, can also be set by env IPL_PRINTER (net port)")
	PrinterAddressPort = flag.String("port", os.Getenv("IPL_PORT"), "Specify printer, can also be set by env IPL_PORT (usb port)")

	PrinterAddressType = flag.String("ctype", os.Getenv("IPL_CTYPE"), "Specify printer connection type, can also be set by env IPL_CTYPE")

	OptVerbose = flag.Bool("verbose", false, "toggle verbose logging")
	OptBeep    = flag.Bool("beep", true, "toggle connection-beep")
	OptDryRun  = flag.Bool("dry-run", false, "disables connection to printer; for testing")
)

var printer *fp.Printer

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	flag.Parse()
	conf := GetConfig()

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

	gmux.Path("/api/job/{uuid}").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleJobAPI)))

	gmux.Path("/img/{uuid}").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleGetImg)))

	gmux.Path("/job/{uuid}").
		Methods("GET").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handleJob)))

	gmux.Path("/api/print").
		Methods("POST").
		Handler(ErrorHandlerMiddleware(http.HandlerFunc(handlePrintPOST)))

	addr := T(*ListenAddr != "", *ListenAddr, conf.Listen)

	if addr == "" {
		addr = "[::]:8070"
	}

	log.Printf("Listening on %s", addr)
	log.Fatalf("Failed to ListenAndServe: %s",
		http.ListenAndServe(addr, gmux))
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
	uid := uuid.New()
	newImageCh <- uid

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)

	fmt.Fprintf(w, "job id: %s\n", uid)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "Invalid File Upload: " + err.Error(),
			Progress: -1,
			Done:     true,
		}

		return
	}

	log.Printf("[POST] file %d bytes", len(data))

	q := r.URL.Query()

	job := &PrintJob{
		UUID: uid,

		optresize:  len(q["resize"]) > 0,
		optstretch: len(q["stretch"]) > 0,
		optrotate:  len(q["rotate"]) > 0,
		optcenterh: len(q["centerh"]) > 0,
		optcenterv: len(q["centerv"]) > 0,
		opttiling:  len(q["tiling"]) > 0, //TODO: use
	}

	dname := ""
	dnames := q["dither"]

	log.Printf("%+v", q)
	if len(dnames) > 0 {
		dname = dnames[0]
	}

	job.ditherer = DitherFromString(dname)

	job.PFCount = 1
	pfs := q["pf"]
	if len(pfs) > 0 {
		i, err := strconv.ParseUint(pfs[0], 10, 32)
		job.PFCount = uint(i)
		if err != nil {
			imageUpdateCh <- Status{
				UUID:     uid,
				Step:     "Invalid PF Count: " + err.Error(),
				Progress: -1,
				Done:     true,
			}

			return
		}
	}

	sizexs, sizeys := q["x"], q["y"]
	if len(sizexs) == 0 || len(sizeys) == 0 {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "No Size of Label Specified",
			Progress: -1,
			Done:     true,
		}

		return
	}

	x64, err := strconv.ParseUint(sizexs[0], 10, 31)
	if err != nil {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "Invalid width: " + err.Error(),
			Progress: -1,
			Done:     true,
		}
		return
	}

	y64, err := strconv.ParseUint(sizeys[0], 10, 31)
	if err != nil {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "Invalid height: " + err.Error(),
			Progress: -1,
			Done:     true,
		}
		return
	}

	job.LabelSize = image.Pt(int(x64), int(y64))

	img, ifmt, err := image.Decode(bytes.NewBuffer(data))
	if err != nil {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "Invalid Image: " + err.Error(),
			Progress: -1,
			Done:     true,
		}

		return
	}

	job.Img = img

	select {
	case printQ <- job:
		break

	default:
		imageUpdateCh <- Status{
			UUID: uid,

			Step:     "print queue full",
			Progress: -1,
			Done:     true,
		}
	}
	log.Printf("[POST] Received Image in %s format bounds: %+v", ifmt, img.Bounds())
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
		if *OptVerbose {
			log.Print("stacktrace from panic: \n" + string(debug.Stack()))
		}

		switch w.Header().Get("Content-Type") {
		case "application/json":
			w.Header().Set("Location", "/")

			w.WriteHeader(500)
			fmt.Fprintf(w, "{\"done\":true,\"message\":\"%s\",\"error\":true}", err)
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

type JobStatusResponse struct {
	Message  string  `json:"message"`
	Progress float32 `json:"progress"`
	Done     bool    `json:"done"`
	Reload   bool    `json:"reload"`
}

func handleJob(w http.ResponseWriter, r *http.Request) {
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

	err = tStatus.Execute(w, JobStatusPageArgs{
		PrintID:       uid.String(),
		InitialStatus: fmt.Sprintf("%s %.2f%%", status.Step, status.Progress*100),
	})
}

func handleJobAPI(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("[GET] /api/status/%s\n%+v", id, status)

	enc := json.NewEncoder(w)
	err = enc.Encode(&JobStatusResponse{
		Message:  status.Step,
		Progress: status.Progress,

		Done:   status.Done,
		Reload: status.Reload,
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
	  <meta http-equiv="Refresh" content="0; URL=/job/%s" />
	</head>`, uid)

	file, header, err := r.FormFile("file")
	if err != nil {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "Invalid File Upload: " + err.Error(),
			Progress: -1,
			Done:     true,
		}

		return
	}

	defer file.Close()

	log.Printf("[POST] file '%s' %d bytes", header.Filename, header.Size)

	job := &PrintJob{
		UUID: uid,

		ditherer: DitherFromString(r.FormValue("dither")),

		optresize:  BoolFromString(r.FormValue("resize")),
		optstretch: BoolFromString(r.FormValue("stretch")),
		optrotate:  BoolFromString(r.FormValue("rotate")),
		optcenterh: BoolFromString(r.FormValue("centerh")),
		optcenterv: BoolFromString(r.FormValue("centerv")),
		opttiling:  BoolFromString(r.FormValue("tiling")), //TODO: use
	}

	job.PFCount = 1
	if len(r.Form["pf"]) > 0 {
		i, err := strconv.ParseUint(r.FormValue("pf"), 10, 32)
		job.PFCount = uint(i)
		if err != nil {
			imageUpdateCh <- Status{
				UUID:     uid,
				Step:     "Invalid PF Count: " + err.Error(),
				Progress: -1,
				Done:     true,
			}

			return
		}
	}

	sizexs, sizeys := r.FormValue("x"), r.FormValue("y")
	if len(sizexs) == 0 || len(sizeys) == 0 {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "No Size of Label Specified",
			Progress: -1,
			Done:     true,
		}

		return
	}

	x64, err := strconv.ParseUint(sizexs, 10, 32)
	if err != nil {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "Invalid width: " + err.Error(),
			Progress: -1,
			Done:     true,
		}
		return
	}

	y64, err := strconv.ParseUint(sizeys, 10, 32)
	if err != nil {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "Invalid height: " + err.Error(),
			Progress: -1,
			Done:     true,
		}
		return
	}

	job.LabelSize = image.Pt(int(x64), int(y64))

	img, ifmt, err := image.Decode(file)
	if err != nil {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "Invalid Image: " + err.Error(),
			Progress: -1,
			Done:     true,
		}

		return
	}

	job.Img = img

	select {
	case printQ <- job:
		break

	default:
		imageUpdateCh <- Status{
			UUID: uid,

			Step:     "print queue full",
			Progress: -1,
			Done:     true,
		}
	}
	log.Printf("[POST] Received Image in %s format bounds: %+v", ifmt, img.Bounds())
}

func b64image(img image.Image) string {
	b := &bytes.Buffer{}
	err := png.Encode(b, img)
	if err != nil {
		panic(err)
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

type JobStatusPageArgs struct {
	InitialStatus string
	PrintID       string
}

func handleGetImg(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	conf := GetConfig()

	v := mux.Vars(r)
	t, ok := v["uuid"]
	if !ok {
		panic("no UUID")
	}

	uid, err := uuid.Parse(t)
	if err != nil {
		panic(err)
	}

	log.Printf("Serving %s", conf.Saves+uid.String()+".png")
	http.ServeFile(w, r, conf.Saves+uid.String()+".png")
}

func T[K any](c bool, a, b K) K {
	if c {
		return a
	}

	return b
}
