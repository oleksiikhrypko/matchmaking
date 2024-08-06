package service

import (
	"container/list"
	"context"
	"log"
	"sync"
	"time"
)

// Controller is a struct that manages sessions and rooms
type Controller struct {
	ctx   context.Context
	lists map[string]*list.List

	keyLockers map[string]*sync.Mutex

	chDone        chan *WaitRoom
	chStopped     chan struct{}
	onRoomReadyFn OnRoomReadyFn

	shutdownTimeout time.Duration
	lockerA         sync.Mutex
	lockerB         sync.Mutex
}

// OnRoomReadyFn is a function that is called when there is a room that is ready to start the game
type OnRoomReadyFn func(sessions []Session, room *WaitRoom)

func NewController(ctx context.Context, onRoomReadyFn OnRoomReadyFn) *Controller {
	s := &Controller{
		ctx:             ctx,
		lockerA:         sync.Mutex{},
		lockerB:         sync.Mutex{},
		keyLockers:      make(map[string]*sync.Mutex), // key -> locker
		lists:           make(map[string]*list.List),  // key -> list
		chDone:          make(chan *WaitRoom),         // channel to notify that room is ready
		chStopped:       make(chan struct{}),          // channel to notify that controller is stopped
		onRoomReadyFn:   onRoomReadyFn,                // function to call when room is ready
		shutdownTimeout: 1 * time.Second,
	}
	go s.doWaitRoom()
	go s.onCancel()

	return s
}

func (c *Controller) getKeyLocker(key string) *sync.Mutex {
	c.lockerA.Lock()
	defer c.lockerA.Unlock()
	locker, ok := c.keyLockers[key]
	if !ok {
		locker = &sync.Mutex{}
		c.keyLockers[key] = locker
	}
	return locker
}

func (c *Controller) getKeyList(key string) *list.List {
	c.lockerB.Lock()
	defer c.lockerB.Unlock()
	l, ok := c.lists[key]
	if !ok {
		l = list.New()
		c.lists[key] = l
	}
	return l
}

// AddSessionToRoom adds session to room with key and equal config
func (c *Controller) AddSessionToRoom(key string, sess Session, roomCfg WaitRoomConfig) {
	locker := c.getKeyLocker(key)
	locker.Lock()
	defer locker.Unlock()

	var (
		ok bool
		l  = c.getKeyList(key)
	)

	// try to add to existing room first
	var wr *WaitRoom
	for e := l.Front(); e != nil; e = e.Next() {
		if wr, ok = e.Value.(*WaitRoom); !ok {
			continue
		}
		// check if room has required config
		if !wr.GetConfig().IsEqual(roomCfg) {
			continue
		}
		// try to add session to room
		if added := wr.Add(sess); added {
			return
		}
	}

	// no free room with config, create new
	wr = NewWaitRoom(c.ctx, roomCfg, c.chDone)
	wr.Add(sess)

	el := l.PushBack(wr)

	wr.OnBeforeDone(func() {
		locker.Lock()
		defer locker.Unlock()
		l.Remove(el)
	})
}

func (c *Controller) doWaitRoom() {
	for room := range c.chDone {
		c.onRoomReadyFn(room.GetSessions(), room)
	}
	select {
	case c.chStopped <- struct{}{}:
	default:
	}
}

func (c *Controller) onCancel() {
	<-c.ctx.Done()
	time.Sleep(c.shutdownTimeout)

	close(c.chDone)

	// let's see how many rooms we have in the lists
	c.lockerB.Lock()
	defer c.lockerB.Unlock()
	for k, v := range c.lists {
		log.Printf("room key: %s, rooms: %d\n", k, v.Len())
	}
}

// Wait waits until controller is stopped
func (c *Controller) Wait() {
	<-c.chStopped
	close(c.chStopped)
}
