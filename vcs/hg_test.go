package vcs

import (
	"bytes"
	"reflect"
	"testing"
)

const desiredWC = "/wc"

var magicHgRev = mercurialRev{0x0d, 0x9c, 0x2b, 0x3c, 0x7b, 0xce, 0x68, 0xef, 0x99, 0x50, 0xd2, 0x37, 0xea, 0xc5, 0xff, 0x67, 0xf1, 0x17, 0xbf, 0xf5}

func newIsolatedMercurialWC(path string, c mockCommander) *mercurialWC {
	return &mercurialWC{
		hg:   &Mercurial{Program: "hg", commander: &c},
		path: path,
	}
}

func TestMercurialCurrent(t *testing.T) {
	mc := mockCommander{
		{Out: *bytes.NewBufferString("0d9c2b3c7bce68ef9950d237eac5ff67f117bff5\n")},
	}
	wc := newIsolatedMercurialWC(desiredWC, mc)
	rev, err := wc.Current()
	if err != nil {
		t.Errorf("wc.Current() error: %v", err)
	}
	if r := magicHgRev; rev != r {
		t.Errorf("wc.Current() = %v; want %v", rev.Rev(), r.Rev())
	}
	if d := mc[0].Dir; d != desiredWC {
		t.Errorf("cd = %v; want %v", d, desiredWC)
	}
	if args, want := mc[0].Args, ([]string{"hg", "identify", "--debug", "-i"}); !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v; want %v", args, want)
	}
}

func TestParseIdentifyOutput(t *testing.T) {
	tests := []struct {
		Arg   string
		Rev   mercurialRev
		Error bool
	}{
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5", magicHgRev, false},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5\n", magicHgRev, false},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5+", magicHgRev, false},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5+\n", magicHgRev, false},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff", mercurialRev{}, true},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff\n", mercurialRev{}, true},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff50", mercurialRev{}, true},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff50\n", mercurialRev{}, true},
	}
	for _, test := range tests {
		rev, err := parseIdentifyOutput([]byte(test.Arg))
		if err != nil && !test.Error {
			t.Errorf("parseIdentifyOutput(%q) error: %v", test.Arg, err)
		} else if err == nil && test.Error {
			t.Errorf("parseIdentifyOutput(%q) expected an error", test.Arg)
		}
		if rev != test.Rev {
			t.Errorf("parseIdentifyOutput(%q) = %v; want %v", test.Arg, rev.Rev(), test.Rev.Rev())
		}
	}
}
