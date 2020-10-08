// Note: a lot of this code was copied from PaceDev's OTO framework.
package main

import (
	"fmt"
	"go/token"
	"go/types"
	"log"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

var errNotFound = errors.New("not found")

type definition struct {
	PackageName string            `json:"packageName,omitempty"`
	Services    []service         `json:"services,omitempty"`
	Objects     []object          `json:"objects,omitempty"`
	Imports     map[string]string `json:"imports,omitempty"`
}

// Object looks up an object by name. Returns errNotFound error
// if it cannot find it.
func (d *definition) Object(name string) (*object, error) {
	for i := range d.Objects {
		obj := &d.Objects[i]
		if obj.Name == name {
			return obj, nil
		}
	}
	return nil, errNotFound
}

type service struct {
	Name    string   `json:"name,omitempty"`
	Methods []method `json:"methods,omitempty"`
}

type method struct {
	Name         string    `json:"name,omitempty"`
	InputObject  fieldType `json:"inputObject,omitempty"`
	OutputObject fieldType `json:"outputObject,omitempty"`
}

type object struct {
	TypeID   string  `json:"typeID"`
	Name     string  `json:"name"`
	Imported bool    `json:"imported,omitempty"`
	Fields   []field `json:"fields,omitempty"`
}

type field struct {
	Name       string    `json:"name,omitempty"`
	Type       fieldType `json:"type,omitempty"`
	OmitEmpty  bool      `json:"omitEmpty,omitempty"`
	Conversion string
}

type fieldType struct {
	TypeID         string `json:"typeID"`
	TypeName       string `json:"typeName"`
	Multiple       bool   `json:"multiple"`
	Package        string `json:"package"`
	IsObject       bool   `json:"isObject"`
	IsSlice        bool
	SliceElem      string
	IsMap          bool
	MapKey         string
	MapElem        string
	TypeConversion string
}

func (f fieldType) JSType() (string, error) {
	if f.IsObject {
		return "object", nil
	}
	switch f.TypeName {
	case "interface{}":
		return "any", nil
	case "map[string]interface{}":
		return "object", nil
	case "string":
		return "string", nil
	case "bool":
		return "boolean", nil
	case "int", "int16", "int32", "int64",
		"uint", "uint16", "uint32", "uint64",
		"float32", "float64":
		return "number", nil
	}
	return "", errors.Errorf("gentype: type not supported: %s", f.TypeName)
}

type parser struct {
	Verbose bool

	ExcludeInterfaces []string

	patterns []string
	def      definition

	IncludeTypes map[string]interface{}

	// outputObjects marks output object names.
	outputObjects map[string]struct{}
	// objects marks object names.
	objects map[string]struct{}
}

// newParser makes a fresh parser using the specified patterns.
// The patterns should be the args passed into the tool (after any flags)
// and will be passed to the underlying build system.
func newParser(patterns ...string) *parser {
	return &parser{
		patterns:     patterns,
		IncludeTypes: map[string]interface{}{},
	}
}

func (p *parser) parse() (definition, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedTypes | packages.NeedDeps | packages.NeedName,
		Tests: false,
	}
	p.outputObjects = make(map[string]struct{})
	p.objects = make(map[string]struct{})
	var excludedObjectsTypeIDs []string
	pkgs, err := packages.Load(cfg, p.patterns...)
	if err != nil {
		return p.def, err
	}
	for _, pkg := range pkgs {
		p.def.PackageName = pkg.Name
		if p.Verbose {
			log.Println("Parsing ", pkg.Name, " (", pkg.PkgPath, ")")

		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			switch item := obj.Type().Underlying().(type) {
			// case *types.Interface:
			// 	s, err := p.parseService(pkg, obj, item)
			// 	if err != nil {
			// 		return p.def, err
			// 	}
			// 	if isInSlice(p.ExcludeInterfaces, name) {
			// 		for _, method := range s.Methods {
			// 			excludedObjectsTypeIDs = append(excludedObjectsTypeIDs, method.InputObject.TypeID)
			// 			excludedObjectsTypeIDs = append(excludedObjectsTypeIDs, method.OutputObject.TypeID)
			// 		}
			// 		continue
			// 	}
			// 	p.def.Services = append(p.def.Services, s)
			case *types.Struct:
				err := p.parseObject(pkg, obj, item)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
	// remove any excluded objects
	nonExcludedObjects := make([]object, 0, len(p.def.Objects))
	for _, object := range p.def.Objects {
		excluded := false
		for _, excludedTypeID := range excludedObjectsTypeIDs {
			if object.TypeID == excludedTypeID {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}
		nonExcludedObjects = append(nonExcludedObjects, object)
	}
	p.def.Objects = nonExcludedObjects
	sort.Slice(p.def.Services, func(i, j int) bool {
		return p.def.Services[i].Name < p.def.Services[j].Name
	})
	if err := p.addOutputFields(); err != nil {
		return p.def, err
	}
	return p.def, nil
}

// func (p *parser) parseService(pkg *packages.Package, obj types.Object, interfaceType *types.Interface) (service, error) {
// 	var s service
// 	s.Name = obj.Name()
// 	if p.Verbose {
// 		fmt.Printf("%s ", s.Name)
// 	}
// 	l := interfaceType.NumMethods()
// 	for i := 0; i < l; i++ {
// 		m := interfaceType.Method(i)
// 		method, err := p.parseMethod(pkg, s.Name, m)
// 		if err != nil {
// 			return s, err
// 		}
// 		s.Methods = append(s.Methods, method)
// 	}
// 	return s, nil
// }

// func (p *parser) parseMethod(pkg *packages.Package, serviceName string, methodType *types.Func) (method, error) {
// 	var m method
// 	m.Name = methodType.Name()
// 	sig := methodType.Type().(*types.Signature)
// 	inputParams := sig.Params()
// 	if inputParams.Len() != 1 {
// 		return m, p.wrapErr(errors.New("invalid method signature: expected Method(MethodRequest) MethodResponse"), pkg, methodType.Pos())
// 	}
// 	var err error
// 	m.InputObject, err = p.parseFieldType(pkg, inputParams.At(0))
// 	if err != nil {
// 		return m, errors.Wrap(err, "parse input object type")
// 	}
// 	outputParams := sig.Results()
// 	if outputParams.Len() != 1 {
// 		return m, p.wrapErr(errors.New("invalid method signature: expected Method(MethodRequest) MethodResponse"), pkg, methodType.Pos())
// 	}
// 	m.OutputObject, err = p.parseFieldType(pkg, outputParams.At(0))
// 	if err != nil {
// 		return m, errors.Wrap(err, "parse output object type")
// 	}
// 	p.outputObjects[m.OutputObject.TypeName] = struct{}{}
// 	return m, nil
// }

// parseObject parses a struct type and adds it to the definition.
func (p *parser) parseObject(pkg *packages.Package, o types.Object, v *types.Struct) error {
	var obj object
	obj.Name = o.Name()

	if _, ok := p.IncludeTypes[obj.Name]; !ok {
		if p.Verbose {
			log.Println(" Skipping ", obj.Name)
		}
		return nil
	}
	if p.Verbose {
		log.Println(" Including ", obj.Name)
	}
	if _, found := p.objects[obj.Name]; found {
		// if this has already been parsed, skip it
		return nil
	}
	if o.Pkg().Name() != pkg.Name {
		obj.Imported = true
	}
	typ := v.Underlying()
	st, ok := typ.(*types.Struct)
	if !ok {
		return p.wrapErr(errors.New(obj.Name+" must be a struct"), pkg, o.Pos())
	}
	obj.TypeID = o.Pkg().Path() + "." + obj.Name
	for i := 0; i < st.NumFields(); i++ {
		//		fmt.Println("v: field", st.Field(i).Type())
		field, err := p.parseField(pkg, st.Field(i))
		if err != nil {
			return err
		}
		//		fmt.Println("v: After parsefield", field.Type)
		obj.Fields = append(obj.Fields, field)
	}
	p.def.Objects = append(p.def.Objects, obj)
	p.objects[obj.Name] = struct{}{}
	return nil
}

// Values to be used in the Get(index ref.Val) func of the struct
func CELTypeConversionGet(ft fieldType) string {

	// Check if we know this type
	switch ft.TypeName {
	case "int":
		return `types.Int(v.{{ .Name }})`
	case "float64":
		return `types.Double(v.{{ .Name }})`
	case "string":
		return `types.String(v.{{ .Name }})`
	case "time.Time":
		return `types.Timestamp{timestamppb.New(v.{{ .Name }})}`
	}

	// Check if it's a slice
	if ft.IsSlice {
		return `types.NewDynamicList(attributeProvider{}, v.{{ .Name}})`
	}

	if ft.IsMap {
		return `types.NewDynamicMap(attributeProvider{}, v.{{ .Name}})`
	}
	return ""
}

func (p *parser) parseField(pkg *packages.Package, v *types.Var) (field, error) {
	var f field
	f.Name = v.Name()
	if !v.Exported() {
		return f, p.wrapErr(errors.New(f.Name+" must be exported"), pkg, v.Pos())
	}
	var err error
	f.Type, err = p.parseFieldType(pkg, v)
	//fmt.Println("v: Done with fType: ", f.Type)
	f.Conversion = CELTypeConversionGet(f.Type)
	if err != nil {
		return f, errors.Wrap(err, "parse type")
	}
	return f, nil
}

func (p *parser) parseFieldType(pkg *packages.Package, obj types.Object) (fieldType, error) {
	var ftype fieldType
	pkgPath := pkg.PkgPath
	resolver := func(other *types.Package) string {
		if other.Name() != pkg.Name {
			if p.def.Imports == nil {
				p.def.Imports = make(map[string]string)
			}
			if other.Name() != "time" {
				p.def.Imports[other.Path()] = other.Name()
			}
			ftype.Package = other.Path()
			pkgPath = other.Path()
			return other.Name()
		}
		return "" // no package prefix
	}

	ftype.Package = pkg.Name
	typ := obj.Type()
	if slice, ok := obj.Type().(*types.Slice); ok {

		ftype.SliceElem = getLocalTypeName(slice.Elem().String(), ftype.Package)
		ftype.Multiple = true
		ftype.IsSlice = true

	}
	if maptype, ok := obj.Type().(*types.Map); ok {
		typ = obj.Type()
		ftype.Multiple = true
		ftype.IsMap = true
		ftype.MapKey = getLocalTypeName(maptype.Key().String(), ftype.Package)
		ftype.MapElem = getLocalTypeName(maptype.Elem().String(), ftype.Package)
	}

	if named, ok := typ.(*types.Named); ok {
		if structure, ok := named.Underlying().(*types.Struct); ok {
			if err := p.parseObject(pkg, named.Obj(), structure); err != nil {
				return ftype, err
			}
			ftype.IsObject = true
		}
	}

	ftype.TypeName = types.TypeString(typ, resolver)
	typeNameWithoutPackage := types.TypeString(typ, func(other *types.Package) string { return "" })
	ftype.TypeID = pkgPath + "." + typeNameWithoutPackage
	return ftype, nil
}

// Since our generated code will live in this package, remove package references from all local types,
// and leave the package name on imported types.
// E.g.,  when called with
//    fullName = github.com/ezachrisen/indigo/cel/examples.Grade
//    localPackage = examples
// return
//    Grade
func getLocalTypeName(fullName string, localPackage string) string {

	partsSlash := strings.Split(fullName, "/") // github.com/ezachrisen/indigo/cel/examples.Grade -> [github.com, ezachrisen, ...]
	lastPart := partsSlash[len(partsSlash)-1]  // examples.Grade
	partsDot := strings.Split(lastPart, ".")   // [examples, Grade]

	if len(partsDot) < 2 { // [Grade] or []
		return fullName
	}

	if partsDot[0] == localPackage { // [0] == examples
		return partsDot[1] // Grade
	}

	return lastPart // examples.Grade
}

// addOutputFields adds built-in fields to the response objects
// mentioned in p.outputObjects.
func (p *parser) addOutputFields() error {
	errorField := field{
		OmitEmpty: true,
		Name:      "Error",
		Type: fieldType{
			TypeName: "string",
		},
	}
	for typeName := range p.outputObjects {
		obj, err := p.def.Object(typeName)
		if err != nil {
			// skip if we can't find it - it must be excluded
			continue
		}
		obj.Fields = append(obj.Fields, errorField)
	}
	return nil
}

func (p *parser) wrapErr(err error, pkg *packages.Package, pos token.Pos) error {
	position := pkg.Fset.Position(pos)
	return errors.Wrap(err, position.String())
}

func isInSlice(slice []string, s string) bool {
	for i := range slice {
		if slice[i] == s {
			return true
		}
	}
	return false
}