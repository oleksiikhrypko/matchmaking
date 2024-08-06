package service

import (
	"context"
	"log"
	"testing"
	"time"
)

func Test_WaitRoom(t *testing.T) {
	log.Println("begin")

	ctx, cancel := context.WithCancel(context.Background())
	chDone := make(chan *WaitRoom)
	wr := NewWaitRoom(ctx, WaitRoomConfig{
		Size:    3,
		MinSize: 2,
		TTL:     2 * time.Second,
	}, chDone)

	go func() {
		time.Sleep(1 * time.Second)
		log.Println("add 1", wr.Add(Session{ID: "1"}))
		// time.Sleep(1 * time.Second)
		log.Println("add 2", wr.Add(Session{ID: "2"}))
		time.Sleep(1 * time.Second)
		log.Println("add 3", wr.Add(Session{ID: "3"}))
	}()

	go func() {
		for v := range chDone {
			log.Println("done: ", v.GetSessions(), v.IsReady())
		}

	}()

	time.Sleep(4 * time.Second)
	cancel()
	close(chDone)
	log.Println("end")
}
