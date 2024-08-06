package service

import (
	"fmt"
	"strings"
)

type Rule struct {
	ConditionsOrder []Param
	ConditionFns    map[Param]func(input int) (key string, ok bool)
}

var RuleErr = fmt.Errorf("rule error")
var RequestErr = fmt.Errorf("request error")

func (r *Rule) BuildRequestRuleKey(req Request) (string, error) {
	keys := make([]string, 0, len(r.ConditionFns))
	for _, attr := range r.ConditionsOrder {
		// check if the attribute exists in the request
		v, ok := req.Attrs[attr]
		if !ok {
			return "", RequestErr
		}

		// check if the attribute has a condition function
		fn, ok := r.ConditionFns[attr]
		if !ok {
			// use the default key function
			keys = append(keys, defaultKeyFn(attr, v))
			continue
		}

		// get the key from the condition function && if the condition function returns false, return an error
		key, ok := fn(v)
		if !ok {
			return "", RequestErr
		}
		keys = append(keys, key)
	}

	return strings.Join(keys, "|"), nil
}

func defaultKeyFn(key Param, input int) string {
	return fmt.Sprintf("%s:%d", key, input)
}

var RuleA = Rule{
	ConditionsOrder: []Param{ParamTable, ParamLeague, ParamLevel, ParamGame},
	ConditionFns: map[Param]func(int) (string, bool){
		ParamTable: func(input int) (string, bool) {
			if input != 7 {
				return "", false
			}
			return fmt.Sprintf("table:%d", input), true
		},
		ParamLeague: func(input int) (string, bool) {
			var (
				minV = 0
				maxV = 3
			)
			if minV < input && input < maxV {
				return fmt.Sprintf("league:%d..%d", minV, maxV), true
			}
			return "", false
		},
		ParamLevel: func(input int) (string, bool) {
			from := input - input*10/100
			to := input + input*10/100

			return fmt.Sprintf("level:%d..%d", from, to), true
		},
	},
}
