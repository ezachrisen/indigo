package main

var src = `
// Code generated DO NOT EDIT
// Generated by the Indigo gentype tool
package {{ .PackageName }}

import (

    "fmt"
    "time"
	"reflect"

    "github.com/ezachrisen/indigo/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
    "google.golang.org/protobuf/types/known/timestamppb"
	
	"github.com/golang/protobuf/ptypes"
	durationpb "github.com/golang/protobuf/ptypes/duration"


	{{ range $key, $value :=  .Imports }}
	{{ $value }} "{{ $key }}"
	{{ end }}
)

// Ensure import sanity
var _ ptypes.DynamicAny 
var _ timestamppb.Timestamp 
var _ time.Time  
var _ durationpb.Duration


{{$packageName:=.PackageName}}

{{ range .Objects }}

    var {{ .Name }}Type = types.NewTypeValue("{{$packageName}}.{{.Name}}", traits.IndexerType)

    func (v {{ .Name }}) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	//log.Println("{{ .Name }}ConvertToNative")
	return nil, fmt.Errorf("cannot convert attribute message to native types")
    }
    
    func (v {{ .Name }}) ConvertToType(typeValue ref.Type) ref.Val {
	//log.Println("{{ .Name }}.ConvertToType")
	return types.NewErr("cannot convert attribute message to CEL types")
    }
    
    func (v {{ .Name }}) Equal(other ref.Val) ref.Val {
	//log.Println("{{ .Name }}.Equal")
	return types.NewErr("attribute message does not support equality")
    }
    
    func (v {{ .Name }}) Type() ref.Type {
	//log.Println("{{ .Name }}.Type")
	return {{ .Name }}Type
    }
    
    func (v {{ .Name }}) Value() interface{} {
	//log.Println("{{ .Name }}.Value")
	return v
    }

    func (v {{ .Name }}) Get(index ref.Val) ref.Val {
	//log.Println("{{ .Name }}.Get ", index, index.Type(), index.Value(), index.Type().TypeName())
	field, ok := index.Value().(string)

	if !ok {
		return types.NewErr("Field %v not found in type %s", index.Value(), "{{ .Name }}")
	}

	switch field {

	{{ range .Fields  }}

	/* Code generation debug information
	    Type: {{ .Type }}
		TNme: {{ .Type.TypeName }}
		Pkg : {{ .Type.Package }}
	    Mult: {{ .Type.Multiple }}
	    ID  : {{ .Type.TypeID }}		
	    Obj : {{ .Type.IsObject }}
        Slc : {{ .Type.IsSlice }}
        Sle : {{ .Type.SliceElem }}
        Map : {{ .Type.IsMap }}
        Mk  : {{ .Type.MapKey }}
        Me  : {{ .Type.MapElem }}
	 */

	    case "{{ .Name }}":
         {{ if .Type.IsSlice }}
              return types.NewDynamicList(cel.AttributeProvider{}, v.{{ .Name}})
         {{ else if .Type.IsMap }}
              return types.NewDynamicMap(cel.AttributeProvider{}, v.{{ .Name}})
	     {{ else if eq .Type.TypeName  "string"  }}
	     	 return types.String(v.{{ .Name }})
	     {{ else if eq .Type.TypeName  "int" }}
	     	 return types.Int(v.{{ .Name }})
	     {{ else if eq .Type.TypeName  "float64" }}
	     	 return types.Double(v.{{ .Name }})
	     {{ else if eq .Type.TypeName  "time.Time" }}
              return types.Timestamp{timestamppb.New(v.{{ .Name }})}
	     {{ else if eq .Type.TypeName  "time.Duration" }}
              return types.Duration{ptypes.DurationProto(v.{{ .Name}})}
         {{ else if .Type.IsObject }}
              return v.{{ .Name }}
	     {{ end }}

         {{ end }} 

         default:
           return nil
       }
    }

	func (v {{ .Name }}) MakeFromFieldMap(fields map[string]ref.Val) cel.CustomType {
		s := {{ .Name }}{}
	
		for fieldName, val := range fields {
			switch fieldName {
			{{ range .Fields }}
			case "{{ .Name }}": 
				{{ if .Type.IsSlice }} 		
					
				 	var ok bool
					var l traits.Lister
					l, ok  = val.(traits.Lister)
					if !ok {
						return nil
					}

					var elemCount int64
					if elemCount, ok = l.Size().Value().(int64); !ok {
						return nil
					}

					for i := int64(0); i < elemCount; i++ {
						elem := l.Get(types.Int(i))
						if convElem, ok := elem.Value().({{.Type.SliceElem}}); ok {
							s.{{.Name}} = append(s.{{.Name}}, convElem)
						} else {
							return nil
						}
					}

				{{ else if .Type.IsMap }}		

					var ok bool
					var l traits.Mapper
					l, ok  = val.(traits.Mapper)
					if !ok {
						return nil
					}

					var elemCount int64
					if elemCount, ok = l.Size().Value().(int64); !ok {
						return nil
					}
					s.{{.Name}} = make(map[{{.Type.MapKey}}]{{.Type.MapElem}}, int(elemCount))
				
					it := l.Iterator()
					for it.HasNext() == types.True {
						key := it.Next()
						var nativeKey {{.Type.MapKey}}
						if nativeKey, ok = key.Value().({{.Type.MapKey}}); !ok {
							return nil
						}
						fieldValue := l.Get(key)
						var nativeFieldValue {{.Type.MapElem}}
						if nativeFieldValue, ok = fieldValue.Value().({{.Type.MapElem}}); !ok {
							return nil
						}
						s.{{.Name}}[nativeKey] = nativeFieldValue
					}
				{{ else if eq .Type.TypeName "int" }}
					if nv, ok := val.Value().(int64); ok {
						s.{{ .Name }} = int(nv)
					} else {
						return nil
					}
				{{ else if eq .Type.TypeName  "time.Duration" }}	
					dur, ok  := val.Value().(*durationpb.Duration)
					if !ok {
						return nil
					}
					timeDur, err := ptypes.Duration(dur)					
					if err != nil {
						return nil
					}
					s.{{ .Name }} = timeDur

				{{ else if eq .Type.TypeName  "time.Time" }}	

					tm, ok  := val.Value().(*timestamppb.Timestamp)
					if !ok {
						return nil
					}
					tmReal, err := ptypes.Timestamp(tm)					
					if err != nil {
						return nil
					}
					s.{{ .Name }} = tmReal

					// timeVal, err := val.ConvertToNative(reflect.TypeOf(s.{{.Name}}))
					// if err != nil {
					// 	return nil
					// }
					// var  ok bool
					// s.{{.Name}}, ok = timeVal.(time.Time)
					// if !ok {
					// 	return nil
					// }
				{{ else }}
					if nv, ok := val.Value().({{.Type.TypeName}}); ok {
						s.{{ .Name }} = nv
					} else {
						return nil
					}		
				{{ end }}
			{{ end }} 	
			default:
				return nil
			}
		}
		return s
	}


    func (v {{ .Name }}) ProvideStructDefintion() cel.StructDefinition {
    	 return cel.StructDefinition{
	   Name: "{{$packageName}}.{{.Name}}",
	   Self: {{.Name}}{},
	   Fields: map[string]*ref.FieldType{
	      {{ range .Fields }}
	      "{{ .Name }}": 
                 {{ if .Type.IsSlice }} 
                    &ref.FieldType{Type: decls.NewListType(decls.Any)},
                 {{ else if .Type.IsMap }}
                    &ref.FieldType{Type: decls.NewMapType(decls.Any, decls.Any)},
	      	     {{ else if eq .Type.TypeName  "string"  }}
	     	     	 &ref.FieldType{Type: decls.String},
	              {{ else if eq .Type.TypeName  "int" }}
	     	         &ref.FieldType{Type: decls.Int},
	              {{ else if eq .Type.TypeName  "float64" }}
     		         &ref.FieldType{Type: decls.Double}, 
	              {{ else if eq .Type.TypeName  "time.Time" }}
     		         &ref.FieldType{Type: decls.Timestamp}, 
	              {{ else if eq .Type.TypeName  "time.Duration" }}
     		         &ref.FieldType{Type: decls.Duration}, 
                 {{ else if .Type.IsObject }}
                    &ref.FieldType{Type: decls.NewObjectType("{{.Type.Package}}.{{.Type.TypeName}}")},
 				 {{ end }}
		   {{ end }}			 
		  },		   			   	   
	   }
    }

{{end}}
`