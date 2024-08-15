package main

import (
	"net/http"

	"bytes"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm/clause"
	"io"
	"io/fs"
	"log"
	"strconv"
	"time"

	"image"
)

type handleFile struct {
	Name string
	FS   fs.FS
}

func (hf *handleFile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, hf.FS, hf.Name)

	return
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

	log.Printf("[POST] received image named '%s' sized %d bytes", header.Filename, header.Size)

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
		Created:     time.Now(),
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

func handlePrintGET(w http.ResponseWriter, r *http.Request) {
	uid := uuid.New()
	newImageCh <- uid

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)

	fmt.Fprintf(w, `<head>
	  <meta http-equiv="Refresh" content="0; URL=/job/%s" />
	</head>`, uid)

	v := r.URL.Query()

	if len(v["uuid"]) == 0 {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "No image UUID specified!",
			Progress: -1,
			Done:     true,
		}

		return
	}

	uuid, err := uuid.Parse(v["uuid"][0])
	if err != nil {
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "Invalid  UUID specified!",
			Progress: -1,
			Done:     true,
		}

		return
	}

	var printfeeds uint = 1
	if len(v["pf"]) > 0 {
		pf, err := strconv.ParseUint(v["pf"][0], 10, 31)
		if err != nil {
			imageUpdateCh <- Status{
				UUID:     uid,
				Step:     "Invalid PF count specified!",
				Progress: -1,
				Done:     true,
			}

			return
		}

		printfeeds = uint(pf)
	}

	log.Printf("[GET] reprint of image %s", uuid)

	job := &PrintJob{
		UUID: uid,

		ditherer: nil,

		public:     false,
		optresize:  false,
		optstretch: false,
		optrotate:  false,
		optcenterh: false,
		optcenterv: false,
		opttiling:  false,
	}

	job.PFCount = printfeeds

	img := GetImage(uuid)
	if img.UUID != uuid { // image not returned
		imageUpdateCh <- Status{
			UUID:     uid,
			Step:     "image not found: " + err.Error(),
			Progress: -1,
			Done:     true,
		}

		return
	}

	imgcfg, imgfmt, err := image.DecodeConfig(bytes.NewReader(img.Data))
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

	job.UnprocessedImage = img

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
	log.Printf("[POST] reprinting %s image with bounds: %d x %d", imgfmt, imgcfg.Width, imgcfg.Height)
}

func handlePrintList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	q := r.URL.Query()

	var (
		offsetstr = q["offset"]
		limitstr  = q["limit"]
	)

	db := GetDB().Model(&Image{})

	var offset, limit uint64
	var err error

	if len(offsetstr) == 0 || len(limitstr) == 0 {
		offset, limit = 0, 12
	} else {
		offset, err = strconv.ParseUint(offsetstr[0], 10, 31)
		if err != nil {
			panic(err)
		}

		limit, err = strconv.ParseUint(limitstr[0], 10, 31)
		if err != nil {
			panic(err)
		}

		if limit > 100 {
			panic("Invalid limit; limit > 100")
		}
	}
	var length int
	db.Select("count(1)").Where("is_processed", false).Where("is_processed", false).Find(&length)

	var l = &ImageList{
		Offset: int(offset),
		Limit:  int(limit),
		Total:  length,
	}
	db.Select("UUID", "UnProcessed", "Processed",
		"IsProcessed", "Ext", "Public", "Name").Where("public", true).Where("is_processed", false).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "created"}, Desc: true}).
		Offset(int(offset)).Limit(int(limit)).Find(&l.Images)

	err = tList.Execute(w, l)
	if err != nil {
		panic(err)
	}
}
