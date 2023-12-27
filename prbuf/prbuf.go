package prbuf

import (
	"encoding/binary"

	// image stuffs
	"bufio"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"

	_ "embed"
	"errors"
	"fmt"
)

var (
	Black = &BW{Black: true}
	White = &BW{Black: false}

	BWModel = color.ModelFunc(bwModel)
)

func bwModel(c color.Color) color.Color {
	r, g, b, _ := c.RGBA()

	// moa than 50% black or sth
	if ((uint64(r) + uint64(g) + uint64(b)) / 3) > (65535 / 2) {
		return Black
	} else {
		return White
	}
}

func IsBlack(c color.Color) bool {
	r, g, b, _ := c.RGBA()

	// moa than 50% black or sth
	return (((uint64(r) + uint64(g) + uint64(b)) / 3) > (65535 / 3 * 2))
}

// for RLE of PRBUF
const MAX_LEN = 127

var (
	ErrInvalidMagicBytes = errors.New("Magic bytes invalid")
)

type ErrPRBUFDecode struct {
	msg string
	err error
}

func (e *ErrPRBUFDecode) Error() string {
	return fmt.Sprintf("prbuf: %s: %s", e.msg, e.err)
}

func prbuferr(msg string, err error) *ErrPRBUFDecode {
	return &ErrPRBUFDecode{
		msg, err,
	}
}

func readU16(r io.Reader) (i uint16, err error) {
	buf := make([]byte, 2)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return
	}

	return binary.BigEndian.Uint16(buf), err
}

func readU8(r io.Reader) (i uint8, err error) {
	buf := make([]byte, 1)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return
	}

	return uint8(buf[0]), err
}

type BW struct {
	Black bool
}

func (bw *BW) RGBA() (r, g, b, a uint32) {
	if bw.Black {
		return 0x0, 0x0, 0x0, 0xffff
	}

	return 0xffff, 0xffff, 0xffff, 0xffff
}

func init() {
	image.RegisterFormat("prbuf",
		string([]byte{0x40, 0x02}),
		Decode,
		DecodeConfig)
}

func DecodeConfig(r io.Reader) (c image.Config, err error) {
	buf := make([]byte, 2)
	i, err := io.ReadFull(r, buf)
	if err != nil {
		err = prbuferr("reading magic", err)

		return
	}

	if i != 2 || buf[0] != 0x40 || buf[1] != 0x02 {
		err = prbuferr("reading magic", ErrInvalidMagicBytes)

		return
	}

	w, err := readU16(r)
	if err != nil {
		err = prbuferr("reading width", err)

		return
	}

	h, err := readU16(r)
	if err != nil {
		err = prbuferr("reading width", err)

		return
	}

	c.Width = int(w)
	c.Height = int(h)
	c.ColorModel = BWModel

	return
}

func Decode(r io.Reader) (img image.Image, err error) { // (img *BWImage, err error) {
	// check magic bytes
	conf, err := DecodeConfig(r)

	iimg := draw.Image(image.NewRGBA(image.Rect(0, 0, conf.Width, conf.Height)))
	img = iimg
	//img = newBW(int(width), int(height))

	var yindex, xindex int
	var color *BW = White
	var b uint8

	for {
		for b == 0 {
			if yindex >= conf.Height {
				// done

				return
			}

			b, err = readU8(r)
			if err != nil {
				err = prbuferr("reading rll data", err)

				return
			}

			if color == Black {
				color = White
			} else {
				color = Black
			}
		}

		iimg.Set(int(xindex), int(yindex), color)

		b--
		xindex++
		if xindex >= conf.Width {
			yindex++
			xindex = 0
			color = White
		}
	}
}

// https://sps-support.honeywell.com/s/article/How-can-the-Fingerprint-PRBUF-command-used-to-print-an-image
func Encode(i image.Image, w io.Writer) {
	bw := bufio.NewWriter(w)
	bw.WriteByte(0x40) // header
	bw.WriteByte(0x02)

	width, height := i.Bounds().Dx(), i.Bounds().Dy()
	var bufs = make([]byte, 2)
	binary.BigEndian.PutUint16(bufs, uint16(width))
	bw.Write(bufs) // width

	binary.BigEndian.PutUint16(bufs, uint16(height))
	bw.Write(bufs) // height

	var state, oldState bool = false, false
	var stateC int

	var b []uint8
	add := func(i byte) {
		b = append(b, i)
	}

	writeRL := func(rl int) {
		//log.Printf("rl: %d", rl)

		for stateC > 0 {
			if stateC > MAX_LEN {
				stateC -= MAX_LEN

				bw.WriteByte(MAX_LEN)
				bw.WriteByte(0x00)
				add(MAX_LEN)
				add(0x00)
			} else {
				l := byte(uint8(stateC))
				bw.WriteByte(l)
				add(l)

				stateC = -1
			}
		}

		oldState = state
	}

	bm, mx := i.Bounds().Min, i.Bounds().Max

	for y := bm.Y; y < mx.Y; y++ {
		for x := bm.X; x < mx.X; x++ {
			c := i.At(x, y)
			state = IsBlack(c)
			if state != oldState {
				// write length of states for this color
				if x == 0 {
					bw.WriteByte(0x00) // change color for first pixaddressen
				}

				writeRL(stateC)
				stateC = 0
			}

			stateC++
			oldState = state
		}

		writeRL(stateC)
		stateC = 0

		state, oldState = false, false

		// ---
		var ttl int
		for _, c := range b {
			ttl += int(c)
		}

		b = make([]uint8, 0)
	}

	// close
	bw.Flush()

	return
}
