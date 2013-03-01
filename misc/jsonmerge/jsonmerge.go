// The jsonmerge command performs an ancestor merge on a JSON object.
package main

import (
	"encoding/json"
	"flag"
)

func main() {
	flag.Parse()
}

func merge(old, a, b interface{}) interface{} {
	if a == nil {
		switch {
		case b == nil:
			return nil
		case old == nil:
			return b
		default:
			return mergeConflict{a, b}
		}
	}
	switch a := a.(type) {
	case bool:
		if b, ok := b.(bool); ok {
			if a == b {
				return a
			} else if old, ok := old.(bool); ok && a == old {
				return b
			} else if ok && b == old {
				return a
			}
		}
		return mergeConflict{a, b}
	case float64:
		if b, ok := b.(float64); ok {
			if a == b {
				return a
			} else if old, ok := old.(float64); ok && a == old {
				return b
			} else if ok && b == old {
				return a
			}
		}
		return mergeConflict{a, b}
	case string:
		if b, ok := b.(string); ok {
			if a == b {
				return a
			} else if old, ok := old.(string); ok && a == old {
				return b
			} else if ok && b == old {
				return a
			}
		}
		return mergeConflict{a, b}
	}
	return nil
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
