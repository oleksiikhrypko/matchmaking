package service

import (
	"context"
	"slices"
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

	beforeDoneFns []func()
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

func (w *WaitRoom) wait() {
	<-w.wrctx.Done()
	w.done()
}

func (w *WaitRoom) done() {
	w.locker.Lock()
	defer w.locker.Unlock()

	for _, fn := range w.beforeDoneFns {
		fn()
	}

	go w.end()
}

func (w *WaitRoom) end() {
	select {
	case <-w.ctx.Done():
	case w.chDone <- w:
	}
}

// Add adds a session to the room
func (w *WaitRoom) Add(sess Session) bool {
	select {
	case <-w.wrctx.Done():
		return false
	default:
	}
	w.locker.Lock()
	defer w.locker.Unlock()

	if len(w.sessions) >= w.conf.Size {
		return false
	}

	w.sessions = append(w.sessions, sess)
	if len(w.sessions) >= w.conf.Size {
		w.closeFn()
	}

	return true
}

func (w *WaitRoom) Remove(sess Session) {
	w.locker.Lock()
	defer w.locker.Unlock()
	for i, s := range w.sessions {
		if s.ID == sess.ID {
			w.sessions = slices.Delete(w.sessions, i, i+1)
			return
		}
	}
}

// IsReady returns true if the room is ready to start the game
func (w *WaitRoom) IsReady() bool {
	w.locker.Lock()
	defer w.locker.Unlock()
	return len(w.sessions) >= w.conf.MinSize
}

// GetSessions returns the sessions in the room
func (w *WaitRoom) GetSessions() []Session {
	w.locker.Lock()
	defer w.locker.Unlock()
	return w.sessions
}

// GetConfig returns the room configuration
func (w *WaitRoom) GetConfig() WaitRoomConfig {
	return w.conf
}

// OnBeforeDone adds a function to be called when room is closed before sending it to the result channel
func (w *WaitRoom) OnBeforeDone(fns ...func()) {
	w.locker.Lock()
	defer w.locker.Unlock()
	w.beforeDoneFns = append(w.beforeDoneFns, fns...)
}
