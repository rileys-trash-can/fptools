package fp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const (
	// Port used by most IPL compatible printers
	DefaultPort = 9100
)

// path should be that of serial device
func OpenPrinter(path string) (p *Printer, err error) {
	var conn *os.File
	conn, err = os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return
	}

	p = &Printer{

		Conn:      conn,
		resReader: bufio.NewReader(conn),
	}

	return p, nil
}

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
	Conn      PrinterConn
	resReader *bufio.Reader
}

type PrinterConn interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
}

func (p *Printer) Read() (res []byte, err error) {
	res, err = p.resReader.ReadBytes('\n')
	if err != nil {
		log.Printf("Rec err: %s", err)
		return
	}

	if len(res) > 0 && res[len(res)-1] == '\n' { // remove \r of \r\n
		res = res[:len(res)-1]
	}

	if len(res) > 0 && res[len(res)-1] == '\r' { // remove \r of \r\n
		res = res[:len(res)-1]
	}

	return res, nil
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

		// empty line lol
		if len(buf) == 0 {
			break
		}

		res.Response = append(res.Response, string(buf))
	}

	// status code
	stat, err := p.Read()
	if err != nil {
		return
	}

	res.Status = string(stat)

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

func (p *Printer) PF(i uint) (err error) {
	err = p.SendCommand(fmt.Sprintf("PF %d", i))
	if err != nil {
		return
	}

	_, err = p.ReadResponse()
	return
}
