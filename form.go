package main

import (
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"bitbucket.org/zombiezen/glados/catalog"
)

const (
	formTag     = "form"
	formOmitted = "-"
)

var formConverts = map[reflect.Type]func(string) (interface{}, error){
	reflect.TypeOf(false):            convertBool,
	reflect.TypeOf(""):               convertString,
	reflect.TypeOf(time.Time{}):      convertTime,
	reflect.TypeOf(catalog.TagSet{}): convertTagSet,
}

func formDecode(dst interface{}, form map[string][]string) error {
	v := reflect.ValueOf(dst).Elem()
	ferr := make(formError)
	for key, val := range form {
		if err := decodeFormValue(v, key, val); err != nil {
			ferr[key] = err
		}
	}
	if len(ferr) > 0 {
		return ferr
	}
	return nil
}

func decodeFormValue(v reflect.Value, key string, val []string) error {
	if key == "" || len(val) == 0 {
		return nil
	}
	t := v.Type()
	for i, n := 0, t.NumField(); i < n; i++ {
		f := t.Field(i)
		if formReflectFieldKey(f) == key {
			if convert := formConverts[f.Type]; convert != nil {
				newVal, err := convert(val[0])
				if err != nil {
					return err
				}
				v.Field(i).Set(reflect.ValueOf(newVal))
			}
			return nil
		}
	}
	return nil
}

func formReflectFieldKey(f reflect.StructField) string {
	switch tag := f.Tag.Get(formTag); tag {
	case "":
		return f.Name
	case "-":
		return ""
	default:
		return tag
	}
	panic("unreachable")
}

func formFieldKey(v, field interface{}) string {
	val, fieldVal := reflect.ValueOf(v).Elem(), reflect.ValueOf(field)
	for i, n := 0, val.NumField(); i < n; i++ {
		f := val.Field(i)
		if f.Addr().Pointer() == fieldVal.Pointer() {
			return formReflectFieldKey(val.Type().Field(i))
		}
	}
	return ""
}

func hasFormField(form map[string][]string, v, field interface{}) bool {
	if k := formFieldKey(v, field); k != "" {
		return len(form[k]) > 0
	}
	return false
}

type formFlag struct {
	Form map[string][]string
	Key  string
}

func (f *formFlag) String() string {
	if v := f.Form[f.Key]; len(v) > 0 {
		return strconv.Quote(v[0])
	}
	return `""`
}

func (f *formFlag) Set(s string) error {
	f.Form[f.Key] = []string{s}
	return nil
}

func addFormFlag(fset *flag.FlagSet, form map[string][]string, key string, usage string) {
	fset.Var(&formFlag{form, key}, key, usage)
}

type formError map[string]error

func (e formError) Error() string {
	msg, n := "", 0
	for _, err := range e {
		if err != nil {
			if n == 0 {
				msg = err.Error()
			}
			n++
		}
	}
	switch n {
	case 0:
		return "0 errors"
	case 1:
		return msg
	case 2:
		return msg + " (and 1 other error)"
	}
	return fmt.Sprintf("%s (and %d other errors)", msg, n-1)
}

func convertBool(s string) (interface{}, error) {
	return strconv.ParseBool(s)
}

func convertString(s string) (interface{}, error) {
	return s, nil
}

func convertTime(s string) (interface{}, error) {
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return v, err
}

func convertTagSet(s string) (interface{}, error) {
	return catalog.ParseTagSet(s), nil
}
