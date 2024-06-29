package main

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"time"
)

var (
	newImageCh       = make(chan uuid.UUID, 1)
	imageUpdateCh    = make(chan Status, 1)
	getImageStatusCh = make(chan StatusReq, 1)
)

type StatusReq struct {
	UUID uuid.UUID

	ResCh chan *Status
}

type Status struct {
	UUID uuid.UUID `json:"jobid,omitempty"`

	Step         string    `json:"message,omitempty"`
	CurrentImage uuid.UUID `json:"image,omitempty"`
	Done         bool      `json:"done,omitempty"`
	Progress     float32   `json:"progress,omitempty"`
	updated      time.Time `json:"-"`
}

func (s *Status) String() string {
	return fmt.Sprintf("%s %.2f%%", s.Step, s.Progress*100)
}

func GetStatus(uuid uuid.UUID) *Status {
	r := StatusReq{}

	r.UUID = uuid
	r.ResCh = make(chan *Status)
	getImageStatusCh <- r

	return <-r.ResCh
}

func init() {
	go doStatus()
}

func doStatus() {
	// move status and job managment to database

	const livetime = time.Hour
	statusMap := make(map[uuid.UUID]*Status)
	t := time.NewTicker(livetime / 2)

	for {

		select {
		case now := <-t.C:
			for k, s := range statusMap {
				if s.updated.Add(livetime).Before(now) {
					log.Printf("[jobclean] removing job %s last updated %s", s.UUID, s.updated)

					delete(statusMap, k)
				}
			}

		case n := <-newImageCh:
			statusMap[n] = &Status{
				UUID: n,

				Step:     "created",
				Progress: 0,
				updated:  time.Now(),
			}

		case update := <-imageUpdateCh:
			update.updated = time.Now()

			statusMap[update.UUID] = &update

		case r := <-getImageStatusCh:
			r.ResCh <- statusMap[r.UUID]
		}
	}
}
