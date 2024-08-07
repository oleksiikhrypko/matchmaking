package service

import (
	"fmt"
	"strings"
)

type Rule struct {
	Params         []Param
	KeyBuilderFns  map[Param]func(input int) (key string)
	MatchRequestFn func(req Request) bool
}

func (r *Rule) MatchRequest(req Request) bool {
	if r.MatchRequestFn == nil {
		return true
	}

	return r.MatchRequestFn(req)
}

func (r *Rule) BuildBucketKey(req Request) string {
	keys := make([]string, 0, len(r.KeyBuilderFns))
	for _, attr := range r.Params {
		// get value if the attribute exists in the request
		v, ok := req.Attrs[attr]
		if !ok {
			continue
		}

		// check if the attribute has a condition function
		fn, ok := r.KeyBuilderFns[attr]
		if !ok {
			keys = append(keys, defaultKeyBuilderFn(attr, v))
			continue
		}

		// get the key from the builder
		if key := fn(v); len(key) != 0 {
			keys = append(keys, key)
		}
	}

	return strings.Join(keys, "|")
}

func defaultKeyBuilderFn(key Param, input int) string {
	return fmt.Sprintf("%s:%d", key, input)
}
