package service

import (
	"context"
	"sync"
	"time"
)

type WaitRoomConfig struct {
	Size    int
	MinSize int
	TTL     time.Duration
}

func (c WaitRoomConfig) IsEqual(other WaitRoomConfig) bool {
	return c.Size == other.Size && c.MinSize == other.MinSize
}

// WaitRoom represents a room where players wait for the game to start
type WaitRoom struct {
	ctx     context.Context
	wrctx   context.Context
	closeFn context.CancelFunc

	sessions []Session
	conf     WaitRoomConfig

	locker sync.Mutex
	chDone chan<- *WaitRoom

	onDoneFns []func()
}

func NewWaitRoom(ctx context.Context, conf WaitRoomConfig, chDone chan<- *WaitRoom) *WaitRoom {
	wrCtx, closeFn := context.WithTimeout(ctx, conf.TTL)
	wr := WaitRoom{
		ctx:     ctx,
		wrctx:   wrCtx,
		closeFn: closeFn,

		conf: conf,

		sessions: make([]Session, 0, conf.Size),

		locker: sync.Mutex{},
		chDone: chDone,
	}
	go wr.wait()
	return &wr
}

func (wr *WaitRoom) wait() {
	<-wr.wrctx.Done()
	wr.done()
}

func (wr *WaitRoom) done() {
	wr.locker.Lock()
	defer wr.locker.Unlock()

	go func() {
		select {
		case <-wr.ctx.Done():
		case wr.chDone <- wr:
		}
	}()

	go func() {
		for _, fn := range wr.onDoneFns {
			fn()
		}
	}()
}

// Add adds a session to the room
func (wr *WaitRoom) Add(sess Session) bool {
	select {
	case <-wr.wrctx.Done():
		return false
	default:
	}
	wr.locker.Lock()
	defer wr.locker.Unlock()

	if len(wr.sessions) >= wr.conf.Size {
		return false
	}

	wr.sessions = append(wr.sessions, sess)
	if len(wr.sessions) >= wr.conf.Size {
		wr.closeFn()
	}

	return true
}

// IsReady returns true if the room is ready to start the game
func (wr *WaitRoom) IsReady() bool {
	wr.locker.Lock()
	defer wr.locker.Unlock()
	return len(wr.sessions) >= wr.conf.MinSize
}

// GetSessions returns the sessions in the room
func (wr *WaitRoom) GetSessions() []Session {
	wr.locker.Lock()
	defer wr.locker.Unlock()
	return wr.sessions
}

// GetConfig returns the room configuration
func (wr *WaitRoom) GetConfig() WaitRoomConfig {
	return wr.conf
}

// OnDone adds a function to be called when room is closed
func (wr *WaitRoom) OnDone(fns ...func()) {
	select {
	case <-wr.wrctx.Done():
		for _, fn := range fns {
			fn()
		}
		return
	default:
		wr.locker.Lock()
		defer wr.locker.Unlock()
		wr.onDoneFns = append(wr.onDoneFns, fns...)
	}
}
