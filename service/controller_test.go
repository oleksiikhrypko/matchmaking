package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
)

func Test_ControllerActions(t *testing.T) {
	beginTime := time.Now()
	log.Println("begin")

	wrCfg4 := WaitRoomConfig{
		Size:    4,
		MinSize: 2,
		TTL:     2 * time.Second,
	}

	wrCfg8 := WaitRoomConfig{
		Size:    8,
		MinSize: 2,
		TTL:     3 * time.Second,
	}

	wrCfg12 := WaitRoomConfig{
		Size:    12,
		MinSize: 2,
		TTL:     5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	var counter int64
	ctrl := NewController(ctx, getOnRoomReadyFn(&counter))

	wg := &sync.WaitGroup{}

	wg.Add(9)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg4, map[Param]int{ParamTable: 7, ParamLeague: 1, ParamLevel: 50}, "A4", 5000)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg8, map[Param]int{ParamTable: 7, ParamLeague: 2, ParamLevel: 60}, "B8", 5000)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg12, map[Param]int{ParamTable: 7, ParamLeague: 2, ParamLevel: 70}, "C12", 5000)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg8, map[Param]int{ParamTable: 7, ParamLeague: 2, ParamLevel: 10}, "A8", 5000)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg12, map[Param]int{ParamTable: 7, ParamLeague: 1, ParamLevel: 20}, "B12", 5000)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg4, map[Param]int{ParamTable: 7, ParamLeague: 2, ParamLevel: 30}, "C4", 5000)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg12, map[Param]int{ParamTable: 7, ParamLeague: 2, ParamLevel: 50}, "A12", 5000)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg4, map[Param]int{ParamTable: 7, ParamLeague: 2, ParamLevel: 50}, "B4", 5000)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg8, map[Param]int{ParamTable: 7, ParamLeague: 1, ParamLevel: 50}, "C8", 5000)
	// wg.Add(1)
	// addSessions(wg, ctx, &RuleA, ctrl, wrCfg4, map[Param]int{ParamTable: 7, ParamLeague: 2, ParamLevel: 50}, "A", 6)
	wg.Wait()
	cancel()
	endTime := time.Now()

	ctrl.Wait()

	log.Printf("end, counter: %d, duration_ms: %d\n", counter, endTime.Sub(beginTime).Milliseconds())
	time.Sleep(1 * time.Second)
}

func getOnRoomReadyFn(counter *int64) func(sessions []Session, room *WaitRoom) {
	return func(sessions []Session, room *WaitRoom) {
		*counter++
		log.Println("room ready", room.IsReady(), sessions)
	}
}

func addSessions(wg *sync.WaitGroup, ctx context.Context, rule *Rule, ctrl *Controller, wrCfg WaitRoomConfig, attrs map[Param]int, group string, count int) {
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		default:
		}
		for i := 1; i <= count; i++ {
			req := Request{
				Session: Session{ID: fmt.Sprintf("group:%s_idx:%d", group, i)},
				Attrs:   attrs,
			}
			key, err := rule.BuildRequestRuleKey(req)
			if err != nil {
				log.Printf("group '%s' idx '%d' rule error\n", group, i)
				continue
			}
			ctrl.AddSessionToRoom(key, req.Session, wrCfg)
		}
	}()
}
