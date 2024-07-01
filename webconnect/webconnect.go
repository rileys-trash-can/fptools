package webconnect

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"image"
	"net/http"
	"net/url"
)

type Dither string

const (
	DitherO4x4         = "o4x4"
	DitherNoise        = "noise"
	DitherBayer Dither = "bayer"
)

type PrintJob struct {
	Image []byte

	PFCount   uint
	LabelSize image.Point
	Ditherer  Dither

	Public  bool
	Resize  bool
	Stretch bool
	Rotate  bool
	Centerh bool
	Centerv bool
	Tiling  bool
}

type PrintJobID struct {
	ID uuid.UUID
}

// Print a printjob at host
// host should only be the host, protocol and additional path part (the /api/print is appended automatically)
func Print(host string, p *PrintJob) (jobid *uuid.UUID, err error) {
	partl, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	u := (&url.URL{
		Scheme: partl.Scheme,
		User:   partl.User,
		Host:   partl.Host,
		Path:   partl.Path,

		// append api call
	}).JoinPath("api", "print")

	v := url.Values{
		//	PFCount   uint
		"pf": []string{fmt.Sprint(p.PFCount)},

		//	LabelSize image.Point
		"x": []string{fmt.Sprint(p.LabelSize.X)},
		"y": []string{fmt.Sprint(p.LabelSize.Y)},

		"ditherer": []string{string(p.Ditherer)},
	}

	es := []string{}
	if p.Public {
		v["public"] = es
	}

	if p.Resize {
		v["resize"] = es
	}

	if p.Stretch {
		v["stretch"] = es
	}

	if p.Rotate {
		v["rotate"] = es
	}

	if p.Centerh {
		v["centerh"] = es
	}

	if p.Centerv {
		v["centerv"] = es
	}

	if p.Tiling {
		v["tiling"] = es
	}

	u.RawQuery = v.Encode()

	h := http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
	}

	http.DefaultClient.Do(&http.Request{
		Method: http.MethodPut,

		URL: u,

		Header: h,

		Body:          readCloser(p.Image),
		ContentLength: int64(len(p.Image)),
	})

	return
}

type rc struct {
	*bytes.Reader
}

func readCloser(b []byte) *rc {
	return &rc{bytes.NewReader(b)}
}

func (b *rc) Close() error {
	b.Reader = nil

	return nil
}
