// The jsonmerge command performs an ancestor merge on a JSON object.
package main

import (
	"encoding/json"
	"flag"
)

func main() {
	flag.Parse()
}

func mergeString(old, a, b string) (merged string, ok bool) {
	switch {
	case a == b:
		return a, true
	case a != b && b == old:
		return a, true
	case a != b && a == old:
		return b, true
	}
	return old, false
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
