package cel

import (
	"fmt"

	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	ex "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type AttributeProvider struct {
	// fallback proto-based type provider
	protos  ref.TypeProvider
	structs map[string]StructDefinition
}

type CustomType interface {
	ProvideStructDefintion() StructDefinition
}

type StructDefinition struct {
	Name   string
	Fields map[string]*ref.FieldType
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
		return decls.NewObjectType(typeName), true
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
	//log.Println("NewValue", typeName, fields)
	return nil
}
