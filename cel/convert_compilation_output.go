package cel

// This files contain functions that convert
//   FROM the CEL compilation output type
//   TO an indigo.Type
//
//

import (
	"errors"
	"fmt" // required by CEL to construct a proto from an expression

	"github.com/ezachrisen/indigo"
	gexpr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// doTypesMatch determines if the indigo and cel types match, meaning
// they can be converted from one to the other
func doTypesMatch(cel *gexpr.Type, igo indigo.Type) error {

	if cel == nil && igo == nil {
		return nil
	}

	if cel == nil {
		return fmt.Errorf("attempt to compare a nil CEL type with indigo type %T", igo)
	}

	if igo == nil {
		return fmt.Errorf("attempt to compare a nil indigo type with a CEL type %T", cel)
	}

	celConverted, err := indigoType(cel)
	if err != nil {
		return err
	}

	if celConverted.String() != igo.String() {
		return fmt.Errorf("type mismatch: CEL: %T (%v), Indigo: %T (%v)", celConverted, celConverted, igo, igo)
	}

	return nil
}

// indigoType convertes from a CEL type to an indigo.Type
func indigoType(t *gexpr.Type) (indigo.Type, error) {

	if t == nil {
		return nil, errors.New("attempt to convert nil to indigo type")
	}

	switch v := t.TypeKind.(type) {
	case *gexpr.Type_MessageType:
		return indigo.ParseType(fmt.Sprintf("proto(%s)", v.MessageType))
	case *gexpr.Type_WellKnown:
		switch v.WellKnown {
		case gexpr.Type_DURATION:
			return indigo.Duration{}, nil
		case gexpr.Type_TIMESTAMP:
			return indigo.Timestamp{}, nil
		default:
			return nil, fmt.Errorf("unknown 'wellknow' type: %T", v)
		}
	case *gexpr.Type_MapType_:
		keyType, err := indigoType(v.MapType.KeyType)
		if err != nil {
			return nil, err
		}

		valType, err := indigoType(v.MapType.ValueType)
		if err != nil {
			return nil, err
		}

		return indigo.Map{
			KeyType:   keyType,
			ValueType: valType,
		}, nil
	case *gexpr.Type_ListType_:
		vType, err := indigoType(v.ListType.ElemType)
		if err != nil {
			return nil, err
		}
		return indigo.List{
			ValueType: vType,
		}, nil
	case *gexpr.Type_Dyn:
		return indigo.Any{}, nil
	case *gexpr.Type_Primitive:
		switch t.GetPrimitive() {
		case gexpr.Type_BOOL:
			return indigo.Bool{}, nil
		case gexpr.Type_DOUBLE:
			return indigo.Float{}, nil
		case gexpr.Type_STRING:
			return indigo.String{}, nil
		case gexpr.Type_INT64:
			return indigo.Int{}, nil
		default:
			return nil, fmt.Errorf("unexpected primitive type %v", v)
		}
	default:
		return nil, fmt.Errorf("unexpected type %v", v)
	}
}
