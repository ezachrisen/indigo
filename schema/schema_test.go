package schema_test

import (
	"reflect"
	"testing"

	"github.com/ezachrisen/indigo/schema"
)

func TestParser(t *testing.T) {

	cases := map[string]struct {
		str       string
		wantError bool
		wantType  schema.Type
	}{
		"int": {
			str:       "int",
			wantError: false,
			wantType:  schema.Int{},
		},
		"float": {
			str:       "float",
			wantError: false,
			wantType:  schema.Float{},
		},
		"map": {
			str:       "map[string]float",
			wantError: false,
			wantType: schema.Map{
				KeyType:   schema.String{},
				ValueType: schema.Float{},
			},
		},
		"list": {
			str:       "[]float",
			wantError: false,
			wantType: schema.List{
				ValueType: schema.Float{},
			},
		},
		"proto": {
			str:       "proto(student)",
			wantError: false,
			wantType: schema.Proto{
				Protoname: "student",
			},
		},
		"proto2": {
			str:       "proto(s)",
			wantError: false,
			wantType: schema.Proto{
				Protoname: "s",
			},
		},
		"proto3": {
			str:       "proto()",
			wantError: true,
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
		typ, err := schema.ParseType(c.str)
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
