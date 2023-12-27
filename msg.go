package fp

import (
	"bufio"
	"bytes"
	"unicode/utf8"
)

const (
	CRLF = "\r\n"
)

// encodes s into utf8
func UTF8encode(s string) []byte {
	buf := make([]byte, 0, len(s))
	r := bufio.NewReader(bytes.NewReader([]byte(s)))

	for {
		r, s, err := r.ReadRune()
		if err != nil || s == 0 {
			break
		}

		rbuf := make([]byte, utf8.UTFMax)
		rbuf = rbuf[:utf8.EncodeRune(rbuf, r)]

		buf = append(buf, rbuf...)
	}

	return buf
}

// readys a message to be send to the computer
// appends CRLF and encodes in utf8
func EncodeMsg(msg string) []byte {
	return UTF8encode(msg + CRLF)
}
