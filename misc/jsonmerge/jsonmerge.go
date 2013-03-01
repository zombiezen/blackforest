// The jsonmerge command performs an ancestor merge on a JSON object.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
)

func main() {
	flag.Parse()
	if flag.NArg() != 3 {
		fmt.Fprintln(os.Stderr, "usage: jsonmerge MYFILE OLDFILE YOURFILE")
		os.Exit(exitUsage)
	}
	objA, err := readJSON(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailure)
	}
	objB, err := readJSON(flag.Arg(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailure)
	}
	objOld, err := readJSON(flag.Arg(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailure)
	}
	mergeObj := merge(objOld, objA, objB)
	if err := json.NewEncoder(os.Stdout).Encode(mergeObj); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailure)
	}
}

func readJSON(path string) (interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var v interface{}
	err = json.NewDecoder(f).Decode(&v)
	return v, err
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
		return &mergeConflict{a, b}
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
		kold, ka, kb := make(StringSet, 0), getStringKeys(va), getStringKeys(vb)
		if told.Kind() == reflect.Map {
			kold = getStringKeys(vold)
		}
		addA, remA := getAddRemoveKeys(kold, ka)
		addB, remB := getAddRemoveKeys(kold, kb)
		result := make(map[string]interface{})
		for k := range ka.Intersect(kb) {
			result[k] = merge(mapIndex(vold, k), mapIndex(va, k), mapIndex(vb, k))
		}
		for k := range addA.Subtract(kb) {
			result[k] = mapIndex(va, k)
		}
		for k := range addB.Subtract(ka) {
			result[k] = mapIndex(vb, k)
		}
		for k := range remA {
			if _, ok := kb[k]; !ok {
				continue
			}
			oldIdx, bIdx := mapIndex(vold, k), mapIndex(vb, k)
			if !reflect.DeepEqual(oldIdx, bIdx) {
				result[k] = &mergeConflict{nil, bIdx}
			}
		}
		for k := range remB {
			if _, ok := ka[k]; !ok {
				continue
			}
			oldIdx, aIdx := mapIndex(vold, k), mapIndex(va, k)
			if !reflect.DeepEqual(oldIdx, aIdx) {
				result[k] = &mergeConflict{aIdx, nil}
			}
		}
		return result
	}
	return &mergeConflict{a, b}
}

func mapIndex(m reflect.Value, k string) interface{} {
	if m.Type().Kind() != reflect.Map {
		return nil
	}
	v := m.MapIndex(reflect.ValueOf(k))
	if !v.IsValid() {
		return nil
	}
	return v.Interface()
}

func getStringKeys(v reflect.Value) StringSet {
	t := v.Type()
	if t.Key().Kind() != reflect.String {
		panic(errors.New("key type not a string"))
	}
	kv := v.MapKeys()
	k := make(StringSet, len(kv))
	for i := range kv {
		k.Add(kv[i].String())
	}
	return k
}

func getAddRemoveKeys(kold, k StringSet) (added, removed StringSet) {
	return k.Subtract(kold), kold.Subtract(k)
}

type StringSet map[string]struct{}

func NewStringSet(s []string) StringSet {
	set := make(StringSet, len(s))
	for _, ss := range s {
		set.Add(ss)
	}
	return set
}

func (ss StringSet) Add(s string) {
	ss[s] = struct{}{}
}

func (s1 StringSet) Subtract(s2 StringSet) StringSet {
	result := make(StringSet, len(s1))
	for k := range s1 {
		if _, ok := s2[k]; !ok {
			result.Add(k)
		}
	}
	return result
}

func (s1 StringSet) Intersect(s2 StringSet) StringSet {
	var result StringSet
	if len(s1) < len(s2) {
		result = make(StringSet, len(s1))
	} else {
		result = make(StringSet, len(s2))
	}
	for k := range s1 {
		if _, ok := s2[k]; ok {
			result.Add(k)
		}
	}
	return result
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
		leftMarker  = `{"CONFLICT":null,"A":`
		splitMarker = `,"B":`
		rightMarker = `}`
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
