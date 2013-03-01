// The jsonmerge command performs an ancestor merge on a JSON object.
package main

import (
	"encoding/json"
	"flag"
	"reflect"
)

func main() {
	flag.Parse()
}

func merge(old, a, b interface{}) interface{} {
	if reflect.DeepEqual(a, b) {
		return a
	}
	vold, va, vb := reflect.ValueOf(old), reflect.ValueOf(a), reflect.ValueOf(b)
	told, ta, tb := vold.Type(), va.Type(), vb.Type()
	if !isSameType(ta, tb) {
		if reflect.DeepEqual(a, old) {
			return b
		} else if reflect.DeepEqual(b, old) {
			return a
		}
		return mergeConflict{a, b}
	}
	if a == nil {
		return nil
	}
	switch ta.Kind() {
	case reflect.Bool, reflect.String, reflect.Float64:
		if isSameType(ta, told) {
			if a == old {
				return b
			} else if b == old {
				return a
			}
		}
	case reflect.Slice:
		if isSameType(ta, told) {
			if reflect.DeepEqual(a, old) {
				return b
			} else if reflect.DeepEqual(b, old) {
				return a
			}
		}
	case reflect.Map:
		// TODO(light)
	}
	return mergeConflict{a, b}
}

// isSameType reports whether t1 and t2 can be treated as the same type.
func isSameType(t1, t2 reflect.Type) bool {
	return t1.Kind() == t2.Kind()
}

// A mergeConflict is a token inserted into a JSON document that indicates a
// merge conflict.
type mergeConflict struct {
	A, B interface{}
}

func (c *mergeConflict) MarshalJSON() ([]byte, error) {
	const (
		leftMarker  = "<<<<<<<"
		splitMarker = "======="
		rightMarker = ">>>>>>>"
	)
	a, err := json.Marshal(c.A)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(c.B)
	if err != nil {
		return nil, err
	}
	data := make([]byte, 0, len(leftMarker)+len(a)+len(splitMarker)+len(b)+len(rightMarker))
	data = append(data, []byte(leftMarker)...)
	data = append(data, a...)
	data = append(data, []byte(splitMarker)...)
	data = append(data, b...)
	data = append(data, []byte(rightMarker)...)
	return data, nil
}

// Exit codes
const (
	exitSuccess = 0
	exitFailure = 1
	exitUsage   = 64
)
