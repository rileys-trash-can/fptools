package fp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
)

const (
	// Port used by most IPL compatible printers
	DefaultPort = 9100
)

// address has to be specified with port
func DialPrinter(address string) (p *Printer, err error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return
	}

	p = &Printer{
		Conn:      conn,
		resReader: bufio.NewReader(conn),
	}

	return p, nil
}

type Printer struct {
	Conn      net.Conn
	resReader *bufio.Reader
}

func (p *Printer) Read() (res []byte, err error) {
	res, err = p.resReader.ReadBytes('\n')
	if err != nil {
		log.Printf("Rec err: %s", err)
		return
	}

	if len(res) > 0 && res[len(res)-1] == '\r' { // remove \r of \r\n
		res = res[:len(res)-1]
	}

	return
}

func (p *Printer) WriteAll(d []byte) (err error) {
	written := 0
	for written < len(d) {
		n, err := p.Conn.Write(d[written:])
		if err != nil {
			return err
		}

		written += n
	}

	return
}

func (p *Printer) SendCommand(msg string) (err error) {
	println(msg)

	enc := EncodeMsg(msg)
	err = p.WriteAll(enc)

	return nil
}

func (p *Printer) SendRaw(r io.Reader) (err error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return
	}

	return p.WriteAll(b)
}

func (p *Printer) Send(r io.Reader) (err error) {
	br := bufio.NewScanner(r)

	for br.Scan() {
		err = p.SendCommand(br.Text())
		if err != nil {
			return
		}
	}

	return
}

type Response struct {
	Command  string
	Response []string // \r\n split
	Status   string
}

func (r *Response) Error() string {
	return fmt.Sprintf("'%s': '%s'", r.Command, r.Status)
}

func (p *Printer) ReadResponse() (res *Response, err error) {
	res = new(Response)

	// command
	cmd, err := p.Read()
	if err != nil {
		return
	}

	res.Command = string(cmd)

	res.Response = make([]string, 0)
	var buf = make([]byte, 1)

	for {
		buf, err = p.Read()
		if err != nil {
			return
		}

		if !(len(buf) < 2 || buf[0] != 0x0d || buf[1] != 0x0a) {
			break
		}

		res.Response = append(res.Response, string(buf))
	}

	// status code
	stat, err := p.Read()
	if err != nil {
		return
	}
	// log.Printf("stat %x - %s", stat, stat)

	res.Status = string(stat[:len(stat)-2])

	log.Printf("> %s", res.Status)

	if res.Status != "Ok" {
		err = res
	}

	return
}

// CLL [<nexp>]
// if field is -1 (i.e. empty) entire canvas is cleared
func (p *Printer) ClearCanvas(field int) (err error) {
	if field < 0 {
		err = p.SendCommand("CLL")
	} else {
		err = p.SendCommand(fmt.Sprintf("CLL %d", field))

	}
	if err != nil {
		return
	}

	_, err = p.ReadResponse()
	return
}

func (p *Printer) PF() (err error) {
	err = p.SendCommand("PF")
	if err != nil {
		return
	}

	_, err = p.ReadResponse()
	return
}
