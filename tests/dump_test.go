package tests_test

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sanity-io/litter"
)

func Function(string, int) (string, error) {
	return "", nil
}

type BlankStruct struct{}

type BasicStruct struct {
	Public  int
	private int
}

type IntAlias int

type InterfaceStruct struct {
	Ifc interface{}
}

type RecursiveStruct struct {
	Ptr *RecursiveStruct
}

type CustomMap map[string]int

type CustomMultiLineDumper struct {
	Dummy int
}

func (cmld *CustomMultiLineDumper) LitterDump(w io.Writer) {
	_, _ = w.Write([]byte("{\n  multi\n  line\n}"))
}

type CustomSingleLineDumper int

func (csld CustomSingleLineDumper) LitterDump(w io.Writer) {
	_, _ = w.Write([]byte("<custom>"))
}

func TestSdump_primitives(t *testing.T) {
	messages := make(chan string, 3)
	sends := make(chan<- int64, 1)
	receives := make(<-chan uint64)

	runTests(t, "primitives", []interface{}{
		false,
		true,
		7,
		int8(10),
		int16(10),
		int32(10),
		int64(10),
		uint8(10),
		uint16(10),
		uint32(10),
		uint64(10),
		uint(10),
		float32(12.3),
		float64(12.3),
		float32(1.0),
		float64(1.0),
		complex64(12 + 10.5i),
		complex128(-1.2 - 0.1i),
		(func(v int) *int { return &v })(10),
		"string with \"quote\"",
		[]int{1, 2, 3},
		interface{}("hello from interface"),
		BlankStruct{},
		&BlankStruct{},
		BasicStruct{1, 2},
		IntAlias(10),
		(func(v IntAlias) *IntAlias { return &v })(10),
		Function,
		func(arg string) (bool, error) { return false, nil },
		nil,
		interface{}(nil),
		CustomMap{},
		CustomMap(nil),
		messages,
		sends,
		receives,
	})
}

func TestSdump_customDumper(t *testing.T) {
	cmld := CustomMultiLineDumper{Dummy: 1}
	cmld2 := CustomMultiLineDumper{Dummy: 2}
	csld := CustomSingleLineDumper(42)
	csld2 := CustomSingleLineDumper(43)
	runTests(t, "customDumper", map[string]interface{}{
		"v1":  &cmld,
		"v2":  &cmld,
		"v2x": &cmld2,
		"v3":  csld,
		"v4":  &csld,
		"v5":  &csld,
		"v6":  &csld2,
	})
}

func TestSdump_pointerAliasing(t *testing.T) {
	p0 := &RecursiveStruct{Ptr: nil}
	p1 := &RecursiveStruct{Ptr: p0}
	p2 := &RecursiveStruct{}
	p2.Ptr = p2

	runTests(t, "pointerAliasing", []*RecursiveStruct{
		p0,
		p0,
		p1,
		p2,
	})
}

func TestSdump_nilIntefacesInStructs(t *testing.T) {
	p0 := &InterfaceStruct{nil}
	p1 := &InterfaceStruct{p0}

	runTests(t, "nilIntefacesInStructs", []*InterfaceStruct{
		p0,
		p1,
		p0,
		nil,
	})
}

func TestSdump_config(t *testing.T) {
	type options struct {
		Compact           bool
		StripPackageNames bool
		HidePrivateFields bool
		HomePackage       string
		Separator         string
		StrictGo          bool
	}

	opts := options{
		StripPackageNames: false,
		HidePrivateFields: true,
		Separator:         " ",
	}

	data := []any{
		opts,
		&BasicStruct{1, 2},
		Function,
		(func(v int) *int { return &v })(20),
		(func(v IntAlias) *IntAlias { return &v })(20),
		litter.Dump,
		func(s string, i int) (bool, error) { return false, nil },
		time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	runTestWithCfg(t, "config_Compact", &litter.Options{
		Compact: true,
	}, data)
	runTestWithCfg(t, "config_HidePrivateFields", &litter.Options{
		HidePrivateFields: true,
	}, data)
	runTestWithCfg(t, "config_HideZeroValues", &litter.Options{
		HideZeroValues: true,
	}, data)
	runTestWithCfg(t, "config_StripPackageNames", &litter.Options{
		StripPackageNames: true,
	}, data)
	runTestWithCfg(t, "config_HomePackage", &litter.Options{
		HomePackage: "tests_test",
	}, data)
	runTestWithCfg(t, "config_FieldFilter", &litter.Options{
		FieldFilter: func(f reflect.StructField, v reflect.Value) bool {
			return f.Type.Kind() == reflect.String
		},
	}, data)
	runTestWithCfg(t, "config_StrictGo", &litter.Options{
		StrictGo: true,
	}, data)
	runTestWithCfg(t, "config_DumpFunc", &litter.Options{
		DumpFunc: func(v reflect.Value, w io.Writer) bool {
			if !v.CanInterface() {
				return false
			}
			if b, ok := v.Interface().(bool); ok {
				if b {
					io.WriteString(w, `"on"`)
				} else {
					io.WriteString(w, `"off"`)
				}
				return true
			}
			return false
		},
	}, data)
	runTestWithCfg(t, "config_FormatTime", &litter.Options{
		FormatTime: true,
	}, data)

	basic := &BasicStruct{1, 2}
	runTestWithCfg(t, "config_DisablePointerReplacement_simpleReusedStruct", &litter.Options{
		DisablePointerReplacement: true,
	}, []any{basic, basic})
	circular := &RecursiveStruct{}
	circular.Ptr = circular
	runTestWithCfg(t, "config_DisablePointerReplacement_circular", &litter.Options{
		DisablePointerReplacement: true,
	}, circular)
}

func TestSdump_multipleArgs(t *testing.T) {
	value1 := []string{"x", "y"}
	value2 := int32(42)

	runTestWithCfg(t, "multipleArgs_noSeparator", &litter.Options{}, value1, value2)
	runTestWithCfg(t, "multipleArgs_lineBreak", &litter.Options{Separator: "\n"}, value1, value2)
	runTestWithCfg(t, "multipleArgs_separator", &litter.Options{Separator: "***"}, value1, value2)
}

func TestSdump_maps(t *testing.T) {
	runTests(t, "maps", []any{
		map[string]string{
			"hello":          "there",
			"something":      "something something",
			"another string": "indeed",
		},
		map[int]string{
			3: "three",
			1: "one",
			2: "two",
		},
		map[int]*BlankStruct{
			2: {},
		},
	})
}

func TestSdump_RecursiveMaps(t *testing.T) {
	mp := make(map[*RecursiveStruct]*RecursiveStruct)
	k1 := &RecursiveStruct{}
	k1.Ptr = k1
	v1 := &RecursiveStruct{}
	v1.Ptr = v1
	k2 := &RecursiveStruct{}
	k2.Ptr = k2
	v2 := &RecursiveStruct{}
	v2.Ptr = v2
	mp[k1] = v1
	mp[k2] = v2
	runTests(t, "recursive_maps", mp)
}

type unexportedStruct struct {
	x int
}
type StructWithUnexportedType struct {
	unexported unexportedStruct
}

func TestSdump_unexported(t *testing.T) {
	runTests(t, "unexported", StructWithUnexportedType{
		unexported: unexportedStruct{},
	})
}

var standardCfg = litter.Options{}

func runTestWithCfg(t *testing.T, name string, cfg *litter.Options, cases ...any) {
	t.Run(name, func(t *testing.T) {
		fileName := fmt.Sprintf("testdata/%s.dump", name)

		dump := cfg.Sdump(cases...)

		reference, err := os.ReadFile(fileName)
		if os.IsNotExist(err) {
			t.Logf("Note: Test data file %s does not exist, writing it; verify contents!", fileName)
			err := os.WriteFile(fileName, []byte(dump), 0644)
			if err != nil {
				t.Error(err)
			}
			return
		}

		assert.Equal(t, string(reference), dump)
	})
}

func runTests(t *testing.T, name string, cases ...any) {
	runTestWithCfg(t, name, &standardCfg, cases...)
}
