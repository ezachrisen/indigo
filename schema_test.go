package indigo_test

import (
	"reflect"
	"testing"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/testdata/school"
	"github.com/matryer/is"
)

func TestProto(t *testing.T) {
	is := is.New(t)

	// Nil message
	a := indigo.Proto{Message: nil}
	_, e := a.ProtoFullName()
	is.True(e != nil)
	is.Equal(e.Error(), "indigo.Proto.Message is nil")

	// Success
	c := indigo.Proto{Message: &school.Student{}}
	s, e := c.ProtoFullName()
	is.NoErr(e)
	is.Equal(s, "testdata.school.Student")

}

func TestString(t *testing.T) {

	cases := map[string]struct {
		typ     indigo.Type
		wantStr string
	}{
		"int": {
			typ:     indigo.Int{},
			wantStr: "int",
		},
		"map": {
			typ: indigo.Map{
				KeyType:   indigo.String{},
				ValueType: indigo.Int{},
			},
			wantStr: "map[string]int",
		},
		"list": {
			typ: indigo.List{
				ValueType: indigo.Duration{},
			},
			wantStr: "[]duration",
		},
	}

	for key, c := range cases {
		str := c.typ.String()
		if str != c.wantStr {
			t.Errorf("case %s: wanted '%s', got '%s'", key, c.wantStr, str)
		}
	}
}

func TestParser(t *testing.T) {

	cases := map[string]struct {
		str       string
		wantError bool
		wantType  indigo.Type
	}{
		"int": {
			str:       "int",
			wantError: false,
			wantType:  indigo.Int{},
		},
		"float": {
			str:       "float",
			wantError: false,
			wantType:  indigo.Float{},
		},
		"map": {
			str:       "map[string]float",
			wantError: false,
			wantType: indigo.Map{
				KeyType:   indigo.String{},
				ValueType: indigo.Float{},
			},
		},
		"list": {
			str:       "[]float",
			wantError: false,
			wantType: indigo.List{
				ValueType: indigo.Float{},
			},
		},
		"proto": {
			str:       "proto(testdata.school.Student)",
			wantError: false,
			wantType:  indigo.Proto{&school.Student{}},
		},
		"list2": {
			str:       "[]",
			wantError: true,
		},
		"list3": {
			str:       "[]xyz",
			wantError: true,
		},
		"map_fail": {
			str:       "map[]float",
			wantError: true,
		},
		"map_fail0": {
			str:       "map[]xyz",
			wantError: true,
		},
		"map_fail_2": {
			str:       "map",
			wantError: true,
		},
		"map_fail_3": {
			str:       "map[string]",
			wantError: true,
		},
	}

	for key, c := range cases {
		typ, err := indigo.ParseType(c.str)
		if c.wantError && err != nil {
			continue
		}
		if c.wantError && err == nil {
			t.Errorf("case %s: wanted error", key)
		}
		if !c.wantError && err != nil {
			t.Errorf("case %s: didn't want error, got: %v", key, err)
		}
		if reflect.TypeOf(typ) != reflect.TypeOf(c.wantType) {
			t.Errorf("case %s: wanted type %s, got %s", key, c.wantType, typ)
		}

		if !reflect.DeepEqual(typ, c.wantType) {
			t.Errorf("case %s: deep equal fails. Want %+v, got %+v", key, c.wantType, typ)
		}
	}
}
