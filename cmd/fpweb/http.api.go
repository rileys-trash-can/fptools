package main

import (
	"net/http"

	"bytes"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"image"
	"io"
	"log"
	"strconv"
	"time"
)

func handleList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query()

	var (
		optall = len(q["all"]) > 0

		offsetstr = q["offset"]
		limitstr  = q["limit"]

		processed     = q["processed"]
		processedType = false
	)

	if len(processed) > 0 {
		processedType = BoolFromString(processed[0])
	}

	enc := json.NewEncoder(w)

	db := GetDB().Model(&Image{})

	if optall {
		var length int
		const limit = 10

		db.Select("count(1)").Find(&length)
		db = db.Select("UUID", "UnProcessed", "Processed",
			"IsProcessed", "Ext", "Public", "Name")

		if len(processed) > 0 {
			db = db.Where("is_processed", processedType)
		}

		var images []Image

		for i := 0; i < length; i += limit {
			db.Offset(i).Limit(limit).Find(&images)

			for k := 0; k < len(images); k++ {
				err := enc.Encode(images[k])
				if err != nil {
					panic(err)
				}
			}
		}
	} else {
		if len(offsetstr) == 0 || len(limitstr) == 0 {
			panic("Invalid or missing offset or limit!")
		}

		offset, err := strconv.ParseUint(offsetstr[0], 10, 31)
		if err != nil {
			panic(err)
		}

		limit, err := strconv.ParseUint(limitstr[0], 10, 31)
		if err != nil {
			panic(err)
		}

		if limit > 100 {
			panic("Invalid limit; limit > 100")
		}

		var length int
		db.Select("count(1)").Find(&length)

		var l = &ImageList{
			Offset: int(offset),
			Limit:  int(limit),
			Total:  length,
		}
		db.Select("UUID", "UnProcessed", "Processed",
			"IsProcessed", "Ext", "Public", "Name").Offset(int(offset)).Limit(int(limit)).Find(&l.Images)

		err = enc.Encode(&l)
		if err != nil {
			panic(err)
		}
	}
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

	log.Printf("[GET] /api/status/%s", id)
	if *OptVerbose {
		log.Printf("status: %+v", status)
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(status)
	if err != nil {
		panic(err)
	}
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

	if *OptVerbose {
		log.Printf("[GET] Serving image %s", uid.String())
	}

	img := GetImage(uid)

	w.Write(img.Data)
}

type PrintJobID struct {
	ID uuid.UUID
}

func handlePrint(w http.ResponseWriter, r *http.Request) {
	uid := uuid.New()
	newImageCh <- uid

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	e := json.NewEncoder(w)
	e.Encode(&PrintJobID{
		uid,
	})

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

	log.Printf("[POST] received image file; size: %d bytes", len(data))

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
		Name:        first(q["name"], time.Now().Format(time.RFC3339)),
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

	log.Printf("[POST] Received %s Image with bounds: %d x %d", imgfmt, imgcfg.Width, imgcfg.Height)
}

func first[K any](a []K, b K) K {
	if len(a) > 0 {
		return a[0]
	}

	return b
}
