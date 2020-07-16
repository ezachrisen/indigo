// This provides *EXPERIMENTAL* support for native Go structs in CEL expressions.
//
package cel

import (
	"fmt"
	"time"

	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/types/known/timestamppb"

	ex "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type AttributeProvider struct {
	// fallback proto-based type provider
	protos  ref.TypeProvider
	structs map[string]StructDefinition
}

type CustomType interface {
	ProvideStructDefintion() StructDefinition
	MakeFromFieldMap(map[string]ref.Val) CustomType
}

type StructDefinition struct {
	Name   string
	Fields map[string]*ref.FieldType
	Self   CustomType
}

func (ap *AttributeProvider) RegisterType(t CustomType) {
	ap.addStruct(t.ProvideStructDefintion())
}
func NewAttributeProvider() *AttributeProvider {
	return &AttributeProvider{
		structs: make(map[string]StructDefinition),
		protos:  types.NewRegistry(),
	}
}

func (ap *AttributeProvider) addStruct(s StructDefinition) {
	ap.structs[s.Name] = s
}

func IsSet(field interface{}) bool {
	//log.Println("Is set", field)
	return true
}

// ref.TypeProvider
func (ap *AttributeProvider) EnumValue(enumName string) ref.Val {
	//log.Println("Enum value", enumName)
	return ap.protos.EnumValue(enumName)
}

func (ap *AttributeProvider) FindIdent(identName string) (ref.Val, bool) {
	//log.Println("Looking for ident ", identName)
	v, b := ap.protos.FindIdent(identName)
	fmt.Println("  - Ident: ", v, " -b", b)
	return ap.protos.FindIdent(identName)
}

func (ap *AttributeProvider) FindType(typeName string) (*ex.Type, bool) {
	//log.Println("FindType:", typeName)
	if _, ok := ap.structs[typeName]; ok {
		return decls.NewTypeType(decls.NewObjectType(typeName)), true
		//return decls.NewObjectType(typeName), true
	}
	return ap.protos.FindType(typeName)
}

func (ap *AttributeProvider) FindFieldType(messageName string, fieldName string) (*ref.FieldType, bool) {
	if strct, ok := ap.structs[messageName]; ok {
		if fld, ok := strct.Fields[fieldName]; ok {
			return fld, true
		}
	}
	return ap.protos.FindFieldType(messageName, fieldName)
}

func (ap *AttributeProvider) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	sdef := ap.structs[typeName]
	if sdef.Self != nil {
		val := sdef.Self.MakeFromFieldMap(fields)
		if refval, ok := val.(ref.Val); ok {
			return refval
		}
	}
	return nil
}

func (ap AttributeProvider) NativeToValue(value interface{}) ref.Val {

	//	log.Printf("NativeToValue called with value %v of type %T\n", value, value)
	switch v := value.(type) {
	case string:
		return types.String(v)
	case int:
		return types.Int(v)
	case float64:
		return types.Double(v)
	case bool:
		return types.Bool(v)
	case time.Time:
		return types.Timestamp{timestamppb.New(v)}
	default:
		// If it's not one of the native types, attempt to convert it to a ref.val.
		// If this works, the type implements the ref.Val interface, possibly
		// beacuse we defined the type elsewhere.
		if rf, ok := value.(ref.Val); ok {
			return rf
		}
		return nil
	}
}
