package main

import (
	"net/http"

	"bytes"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"io"
	"log"
	"strconv"

	"image"
)

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

		public:     len(q["public"]) > 0,
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

	// image handeling
	imgcfg, imgfmt, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		if err != nil {
			imageUpdateCh <- Status{
				UUID:     uid,
				Step:     "Failed to Decode Image (header): " + err.Error(),
				Progress: -1,
				Done:     true,
			}

			return
		}
	}

	job.UnprocessedImage = Image{
		UUID: uuid.New(),

		IsProcessed: false,
		Ext:         imgfmt,
		Data:        data,
		Public:      job.public,
		Name:        "",
	}

	GetDB().Create(&job.UnprocessedImage)

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
	log.Printf("[POST] Received Image in %s format bounds: %d x %d", imgfmt, imgcfg.Width, imgcfg.Height)
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

	err = tStatus.Execute(w, status)
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

		public:     BoolFromString(r.FormValue("public")),
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

	// image handeling
	data, err := io.ReadAll(file)
	if err != nil {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "Failed to Read Image: " + err.Error(),
			Progress: -1,
			Done:     true,
		}

		return
	}

	imgcfg, imgfmt, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		if err != nil {
			imageUpdateCh <- Status{
				UUID:     uid,
				Step:     "Failed to Decode Image (header): " + err.Error(),
				Progress: -1,
				Done:     true,
			}

			return
		}
	}

	job.UnprocessedImage = Image{
		UUID: uuid.New(),

		IsProcessed: false,
		Ext:         imgfmt,
		Data:        data,
		Public:      job.public,
		Name:        header.Filename,
	}

	GetDB().Create(&job.UnprocessedImage)

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
	log.Printf("[POST] Received Image in %s format bounds: %d x %d", imgfmt, imgcfg.Width, imgcfg.Height)
}
