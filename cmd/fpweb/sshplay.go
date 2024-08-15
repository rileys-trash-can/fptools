package main

import (
	_ "embed"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"net/http"
	"time"
)

//go:embed popcorn.wav
var audioe []byte

var addr string

func initmusic() {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("Failed to Listen: %s", err)
	}

	mux := http.NewServeMux()

	port := l.Addr().(*net.TCPAddr).Port
	log.Printf("listen port: %d", port)
	addr = fmt.Sprintf("http://10.0.0.250:%d/audio", port)

	mux.Handle("/audio", http.HandlerFunc(handleAudio))

	log.Fatalf("Failed to serve: %s", http.Serve(l, mux))
}

func handleAudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/wav")

	for i := 0; i < len(audioe); i += 100 {
		var offset int
		if (i + 100) < len(audioe) {
			offset = i + 100
		} else {
			offset = len(audioe)
		}

		w.Write(audioe[i:offset])
	}
}

// opens ssh conn, plays audio and closes start channel when refrain is playing
// please close endprinting chanel when printing is over
func doaudio() (startprinting, endprinting chan struct{}) {
	conf := GetConfig()

	log.Printf("ssh.Dial")
	c, err := ssh.Dial("tcp", conf.SSHAddr, &ssh.ClientConfig{
		User: conf.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(conf.SSHPass),
		},

		HostKeyCallback: HostKeyCallback(conf.SSHKey),
	})
	if err != nil {
		log.Fatalf("Failed to open ssh to '%s': %s", conf.SSHAddr, err)
	}

	log.Printf("c.NewSession")
	s, err := c.NewSession()
	if err != nil {
		log.Fatalf("Failed to open new session: %s", err)
	}

	//s.Stdout = log.Writer()
	//s.Stderr = log.Writer()

	start := time.Now()

	// init channels
	startprinting, endprinting = make(chan struct{}), make(chan struct{})

	go func() {
		log.Printf("s.Run()")
		go func() {
			err = s.Run(fmt.Sprintf("wget \"%s\" -O /dev/dsp", addr))
			if err != nil {
				log.Printf("Failed to s.Run: %s", err)
			}
		}()

		<-endprinting
		log.Printf("Stop channel closed, waiting additional %s", conf.MusicContPlaying)
		time.Sleep(conf.MusicContPlaying)
		log.Printf("additional time over, closing Session")

		err := s.Signal(ssh.SIGKILL)
		if err != nil {
			log.Printf("Failed stopping music signal: %s", err)
		}
		c.Close()
	}()

	log.Printf("started playing: %v", start)

	go func() {
		time.Sleep(time.Millisecond * 11600)

		log.Printf("Music Refrain strated playing; starting printing")

		close(startprinting)
	}()

	return
}

func HostKeyCallback(sshkey string) func(h string, r net.Addr, k ssh.PublicKey) error {
	return func(h string, r net.Addr, k ssh.PublicKey) error {
		log.Printf("h: %s, addr %v, key %s",
			h, r, k,
		)

		return nil
	}
}
