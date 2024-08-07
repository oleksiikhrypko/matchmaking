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

	var RuleA, RuleB, RuleC Rule

	RuleA = Rule{
		Params: []Param{ParamTable, ParamLeague, ParamLevel, ParamGame},
		KeyBuilderFns: map[Param]func(int) string{
			ParamLeague: func(input int) string {
				// skip parameter if it does not meet the condition
				var (
					minV = 0
					maxV = 3
				)
				if minV < input && input < maxV {
					return "league:ruleA"
				}
				return ""
			},
			ParamLevel: func(input int) string {
				var v string
				switch {
				case input < 10:
					v = "A"
				case input < 40:
					v = "B"
				case input < 80:
					v = "C"
				default:
					v = "D"
				}

				return fmt.Sprintf("lvl:%s", v)
			},
		},
		MatchRequestFn: func(req Request) bool {
			v, ok := req.Attrs[ParamLeague]
			if !ok {
				return false
			}
			if 0 < v && v < 3 {
				return true
			}
			return false
		},
	}

	RuleB = RuleA
	RuleB.MatchRequestFn = func(req Request) bool {
		v, ok := req.Attrs[ParamLeague]
		if !ok {
			return false
		}
		if 3 <= v && v <= 8 {
			return true
		}
		return false
	}

	RuleC = RuleA
	RuleC.MatchRequestFn = func(req Request) bool {
		v, ok := req.Attrs[ParamLeague]
		if !ok {
			return false
		}
		if 8 < v {
			return true
		}
		return false
	}

	ctx, cancel := context.WithCancel(context.Background())

	var counter int64
	ctrl := NewController(ctx, getOnRoomReadyFn(&counter))

	wg := &sync.WaitGroup{}

	wg.Add(9)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg4, map[Param]int{ParamTable: 7, ParamLeague: 1, ParamLevel: 49, ParamGame: 1}, "A4", 5000)
	addSessions(wg, ctx, &RuleB, ctrl, wrCfg8, map[Param]int{ParamTable: 4, ParamLeague: 3, ParamLevel: 52, ParamGame: 1}, "B8", 5000)
	addSessions(wg, ctx, &RuleC, ctrl, wrCfg12, map[Param]int{ParamTable: 5, ParamLeague: 9, ParamLevel: 48, ParamGame: 1}, "C12", 5000)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg8, map[Param]int{ParamTable: 6, ParamLeague: 2, ParamLevel: 13, ParamGame: 1}, "A8", 5000)
	addSessions(wg, ctx, &RuleB, ctrl, wrCfg12, map[Param]int{ParamTable: 7, ParamLeague: 4, ParamLevel: 19, ParamGame: 2}, "B12", 5000)
	addSessions(wg, ctx, &RuleC, ctrl, wrCfg4, map[Param]int{ParamTable: 7, ParamLeague: 10, ParamLevel: 30, ParamGame: 1}, "C4", 5000)
	addSessions(wg, ctx, &RuleA, ctrl, wrCfg12, map[Param]int{ParamTable: 6, ParamLeague: 1, ParamLevel: 52, ParamGame: 1}, "A12", 5000)
	addSessions(wg, ctx, &RuleB, ctrl, wrCfg4, map[Param]int{ParamTable: 5, ParamLeague: 5, ParamLevel: 57, ParamGame: 1}, "B4", 5000)
	addSessions(wg, ctx, &RuleC, ctrl, wrCfg8, map[Param]int{ParamTable: 4, ParamLeague: 11, ParamLevel: 61, ParamGame: 1}, "C8", 5000)
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
		wrCfg.Rule = rule
		for i := 1; i <= count; i++ {
			req := Request{
				Session:     Session{ID: fmt.Sprintf("group:%s_idx:%d", group, i)},
				Attrs:       attrs,
				WaitRoomCfg: wrCfg,
			}
			ctrl.AddSessionToRoom(req)
		}
	}()
}
