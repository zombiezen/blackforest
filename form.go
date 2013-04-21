package main

import (
	"errors"
	"flag"
	"reflect"
	"strconv"
	"time"

	"bitbucket.org/zombiezen/blackforest/catalog"
	"github.com/zombiezen/schema"
)

var errRequiredField = errors.New("required field empty")

var decoder = schema.NewDecoder()

func init() {
	decoder.RegisterErrorConverter(time.Time{}, convertTime)
	decoder.RegisterErrorConverter(catalog.TagSet{}, convertTagSet)
	decoder.RegisterConverter(nullString{}, convertNullString)
}

type nullString struct {
	String string
	Valid  bool
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

func convertTime(s string) (reflect.Value, error) {
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	return reflect.ValueOf(v), err
}

func convertTagSet(s string) (reflect.Value, error) {
	return reflect.ValueOf(catalog.ParseTagSet(s)), nil
}

func convertNullString(s string) reflect.Value {
	return reflect.ValueOf(nullString{String: s, Valid: true})
}

func isFormValueEmpty(form map[string][]string, key string) bool {
	v := form[key]
	if len(v) == 0 {
		return true
	}
	return v[0] == ""
}
