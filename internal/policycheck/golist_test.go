package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseList_TaggedTokens_BuildsRecords(t *testing.T) {
	out := "P example.com/p\x00N p\x00D /module/p\x00F main\x00I fmt\x00I net/http\x00G ü file.go\x00G cgo.go\x00E\x00\nP dep\x00N dep\x00D /dep\x00F dep\x00F localreplace\x00E\x00\nP fmt\x00N fmt\x00D /go/src/fmt\x00F std\x00F dep\x00E\x00\n"
	got, err := parseList([]byte(out))
	want := []listedPackage{
		{importPath: "example.com/p", name: "p", dir: "/module/p", mainModule: true, goFiles: []string{"ü file.go", "cgo.go"}, deps: []string{"fmt", "net/http"}},
		{importPath: "dep", name: "dep", dir: "/dep", depOnly: true, localReplace: true},
		{importPath: "fmt", name: "fmt", dir: "/go/src/fmt", depOnly: true, standard: true},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v err=%v; want %+v", got, err, want)
	}
}

func TestParseList_MalformedStream_Errors(t *testing.T) {
	for _, out := range []string{"P p\x00", "E\x00", "P p\x00P q\x00", "P p\x00X x\x00E\x00", "P p\x00F unknown\x00E\x00", "P p\x00E\x00trailing"} {
		t.Run(out, func(t *testing.T) {
			if got, err := parseList([]byte(out)); err == nil {
				t.Fatalf("got=%+v; want malformed stream error", got)
			}
		})
	}
}

func TestCappedBuffer_OverLimit_Errors(t *testing.T) {
	b := cappedBuffer{limit: 3}
	if n, err := b.Write([]byte("abc")); n != 3 || err != nil {
		t.Fatalf("boundary write=(%d,%v); want (3,nil)", n, err)
	}
	if n, err := b.Write([]byte("d")); n != 0 || !errors.Is(err, errOutputTooLarge) || b.buf.String() != "abc" {
		t.Fatalf("overflow write=(%d,%v) bytes=%q", n, err, b.buf.String())
	}
}
