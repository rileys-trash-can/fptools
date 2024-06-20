package main

import (
	"fmt"
	"github.com/google/uuid"
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
	UUID uuid.UUID

	Step     string
	Reload   bool
	Done     bool
	Progress float32
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
	statusMap := make(map[uuid.UUID]*Status)

	for {
		select {
		case n := <-newImageCh:
			statusMap[n] = &Status{
				UUID: n,

				Step:     "created",
				Progress: 0,
			}

		case update := <-imageUpdateCh:
			statusMap[update.UUID] = &update

		case r := <-getImageStatusCh:
			r.ResCh <- statusMap[r.UUID]
		}
	}
}
