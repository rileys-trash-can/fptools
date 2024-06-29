package main

import (
	"net/http"

	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"log"
	"strconv"
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

		var l = ImageList{
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
