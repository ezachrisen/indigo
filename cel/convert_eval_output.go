package cel

// This file contains functions that convert
//    FROM a CEL evaluation output
//    TO Go types

import (
	"fmt"
	"reflect"

	"github.com/ezachrisen/indigo"
	"github.com/google/cel-go/common/types/ref"
)

// convertDynamicMessageToProto converts a *dynamicpb.Message (represented by ref.Val)
// to the wanted proto type represented as an indigo.Type.
// Fails if the indigo.Type is not a proto, or the conversion to the wanted proto fails.
func convertDynamicMessageToProto(r ref.Val, want indigo.Type) (any, error) {

	msg, ok := want.(indigo.Proto)

	if !ok {
		return nil, fmt.Errorf("CEL returned a protobuf, expected %T", want)
	}

	if msg.Message == nil {
		return nil, fmt.Errorf("expected result protocol buffer is nil")
	}

	pb, err := r.ConvertToNative(reflect.TypeOf(msg.Message))
	if err != nil {
		return nil, err
	}

	return pb, nil
}
