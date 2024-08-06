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
	return c.Size == other.Size && c.MinSize == other.MinSize && c.TTL == other.TTL
}

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

func (w *WaitRoom) IsReady() bool {
	w.locker.Lock()
	defer w.locker.Unlock()
	return len(w.sessions) >= w.conf.MinSize
}

func (w *WaitRoom) GetSessions() []Session {
	w.locker.Lock()
	defer w.locker.Unlock()
	return w.sessions
}

func (w *WaitRoom) GetConfig() WaitRoomConfig {
	return w.conf
}

func (w *WaitRoom) OnBeforeDone(fns ...func()) {
	w.locker.Lock()
	defer w.locker.Unlock()
	w.beforeDoneFns = append(w.beforeDoneFns, fns...)
}
