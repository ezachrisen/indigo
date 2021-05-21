package cel

import (
	"testing"

	"github.com/ezachrisen/indigo"
	gexpr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// Test converting between CEL and Indigo types, and whether expected
// type matches work.
func TestTypeConversion(t *testing.T) {

	cases := []struct {
		celType    *gexpr.Type
		indigoType indigo.Type
		wantError  bool
	}{
		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_WellKnown{
					WellKnown: gexpr.Type_DURATION,
				},
			},
			indigoType: indigo.Duration{},
			wantError:  false,
		},
		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_WellKnown{
					WellKnown: gexpr.Type_TIMESTAMP,
				},
			},
			indigoType: indigo.Timestamp{},
			wantError:  false,
		},

		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_Primitive{
					Primitive: gexpr.Type_BOOL,
				},
			},
			indigoType: indigo.Bool{},
			wantError:  false,
		},
		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_Primitive{
					Primitive: gexpr.Type_DOUBLE,
				},
			},
			indigoType: indigo.Float{},
			wantError:  false,
		},
		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_Primitive{
					Primitive: gexpr.Type_STRING,
				},
			},
			indigoType: indigo.String{},
			wantError:  false,
		},
		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_Primitive{
					Primitive: gexpr.Type_INT64,
				},
			},
			indigoType: indigo.Int{},
			wantError:  false,
		},
		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_Primitive{
					Primitive: gexpr.Type_BOOL,
				},
			},
			indigoType: indigo.String{},
			wantError:  true,
		},
		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_MapType_{
					MapType: &gexpr.Type_MapType{
						KeyType: &gexpr.Type{
							TypeKind: &gexpr.Type_Primitive{
								Primitive: gexpr.Type_STRING,
							},
						},
						ValueType: &gexpr.Type{
							TypeKind: &gexpr.Type_Primitive{
								Primitive: gexpr.Type_STRING,
							},
						},
					},
				},
			},
			indigoType: indigo.Map{
				KeyType:   indigo.String{},
				ValueType: indigo.String{},
			},
			wantError: false,
		},
		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_MapType_{
					MapType: &gexpr.Type_MapType{
						KeyType: &gexpr.Type{
							TypeKind: &gexpr.Type_Primitive{
								Primitive: gexpr.Type_STRING,
							},
						},
						ValueType: &gexpr.Type{
							TypeKind: &gexpr.Type_Primitive{
								Primitive: gexpr.Type_INT64,
							},
						},
					},
				},
			},
			indigoType: indigo.Map{
				KeyType:   indigo.String{},
				ValueType: indigo.String{},
			},
			wantError: true,
		},
		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_ListType_{
					ListType: &gexpr.Type_ListType{
						ElemType: &gexpr.Type{
							TypeKind: &gexpr.Type_Primitive{
								Primitive: gexpr.Type_INT64,
							},
						},
					},
				},
			},
			indigoType: indigo.List{
				ValueType: indigo.String{},
			},
			wantError: true,
		},
		{
			celType: &gexpr.Type{
				TypeKind: &gexpr.Type_ListType_{
					ListType: &gexpr.Type_ListType{
						ElemType: &gexpr.Type{
							TypeKind: &gexpr.Type_Primitive{
								Primitive: gexpr.Type_STRING,
							},
						},
					},
				},
			},
			indigoType: indigo.List{
				ValueType: indigo.String{},
			},
			wantError: false,
		},
	}

	// Check whether the two types match, and if
	// errors are being caught
	for _, c := range cases {

		err := doTypesMatch(c.celType, c.indigoType)
		if err != nil && !c.wantError {
			t.Error(err)
		}

		if err == nil && c.wantError {
			t.Error("wanted error")
		}
	}

	// Check if converting FROM indigo TO cel works
	for _, c := range cases {
		cl, err := convertToDeclaration(c.indigoType)
		if err != nil {
			t.Error(err)
		}

		// If we're looking for a type mismatch, we don't have anything to compare to
		if !c.wantError {
			err := doTypesMatch(cl, c.indigoType)
			if err != nil {
				t.Error(err)
			}
		}
	}

	// Check if converting FROM cel TO indigo works
	for _, c := range cases {
		i, err := indigoType(c.celType)
		if err != nil {
			t.Error(err)
		}

		// If we're looking for a type mismatch, we don't have anything to compare to
		if !c.wantError {
			err := doTypesMatch(c.celType, i)
			if err != nil {
				t.Error(err)
			}
		}
	}

}

// // Test converting between CEL and Indigo types, and whether expected
// // type matches work.
// func TestTypeConversion2(t *testing.T) {

// 	cases := []struct {
// 		celType    ref.Val
// 		indigoType indigo.Type
// 		wantError  bool
// 	}{
// 		{
// 			celType:    ctypes.Bool(true),
// 			indigoType: indigo.Bool{},
// 			wantError:  false,
// 		},
// 		// {
// 		// 	celType:    ctypes.TimestampType ,
// 		// 	indigoType: indigo.Timestamp{},
// 		// 	wantError:  false,
// 		// },
// 		// {
// 		// 	celType:    ctypes.MapType,
// 		// 	indigoType: indigo.Map{},
// 		// 	wantError:  false,
// 		// },

// 		// {
// 		// 	celType: &gexpr.Type{
// 		// 		TypeKind: &gexpr.Type_WellKnown{
// 		// 			WellKnown: gexpr.Type_TIMESTAMP,
// 		// 		},
// 		// 	},
// 		// 	indigoType: indigo.Timestamp{},
// 		// 	wantError:  false,
// 		// },

// 		// {
// 		// 	celType: &gexpr.Type{
// 		// 		TypeKind: &gexpr.Type_Primitive{
// 		// 			Primitive: gexpr.Type_BOOL,
// 		// 		},
// 		// 	},
// 		// 	indigoType: indigo.Bool{},
// 		// 	wantError:  false,
// 		// },
// 		// {
// 		// 	celType: &gexpr.Type{
// 		// 		TypeKind: &gexpr.Type_Primitive{
// 		// 			Primitive: gexpr.Type_DOUBLE,
// 		// 		},
// 		// 	},
// 		// 	indigoType: indigo.Float{},
// 		// 	wantError:  false,
// 		// },
// 		// {
// 		// 	celType: &gexpr.Type{
// 		// 		TypeKind: &gexpr.Type_Primitive{
// 		// 			Primitive: gexpr.Type_STRING,
// 		// 		},
// 		// 	},
// 		// 	indigoType: indigo.String{},
// 		// 	wantError:  false,
// 		// },
// 		// {
// 		// 	celType: &gexpr.Type{
// 		// 		TypeKind: &gexpr.Type_Primitive{
// 		// 			Primitive: gexpr.Type_INT64,
// 		// 		},
// 		// 	},
// 		// 	indigoType: indigo.Int{},
// 		// 	wantError:  false,
// 		// },
// 		// {
// 		// 	celType: &gexpr.Type{
// 		// 		TypeKind: &gexpr.Type_Primitive{
// 		// 			Primitive: gexpr.Type_BOOL,
// 		// 		},
// 		// 	},
// 		// 	indigoType: indigo.String{},
// 		// 	wantError:  true,
// 		// },
// 		// {
// 		// 	celType: &gexpr.Type{
// 		// 		TypeKind: &gexpr.Type_MapType_{
// 		// 			MapType: &gexpr.Type_MapType{
// 		// 				KeyType: &gexpr.Type{
// 		// 					TypeKind: &gexpr.Type_Primitive{
// 		// 						Primitive: gexpr.Type_STRING,
// 		// 					},
// 		// 				},
// 		// 				ValueType: &gexpr.Type{
// 		// 					TypeKind: &gexpr.Type_Primitive{
// 		// 						Primitive: gexpr.Type_STRING,
// 		// 					},
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 	},
// 		// 	indigoType: indigo.Map{
// 		// 		KeyType:   indigo.String{},
// 		// 		ValueType: indigo.String{},
// 		// 	},
// 		// 	wantError: false,
// 		// },
// 		// {
// 		// 	celType: &gexpr.Type{
// 		// 		TypeKind: &gexpr.Type_MapType_{
// 		// 			MapType: &gexpr.Type_MapType{
// 		// 				KeyType: &gexpr.Type{
// 		// 					TypeKind: &gexpr.Type_Primitive{
// 		// 						Primitive: gexpr.Type_STRING,
// 		// 					},
// 		// 				},
// 		// 				ValueType: &gexpr.Type{
// 		// 					TypeKind: &gexpr.Type_Primitive{
// 		// 						Primitive: gexpr.Type_INT64,
// 		// 					},
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 	},
// 		// 	indigoType: indigo.Map{
// 		// 		KeyType:   indigo.String{},
// 		// 		ValueType: indigo.String{},
// 		// 	},
// 		// 	wantError: true,
// 		// },
// 		// {
// 		// 	celType: &gexpr.Type{
// 		// 		TypeKind: &gexpr.Type_ListType_{
// 		// 			ListType: &gexpr.Type_ListType{
// 		// 				ElemType: &gexpr.Type{
// 		// 					TypeKind: &gexpr.Type_Primitive{
// 		// 						Primitive: gexpr.Type_INT64,
// 		// 					},
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 	},
// 		// 	indigoType: indigo.List{
// 		// 		ValueType: indigo.String{},
// 		// 	},
// 		// 	wantError: true,
// 		// },
// 		// {
// 		// 	celType: &gexpr.Type{
// 		// 		TypeKind: &gexpr.Type_ListType_{
// 		// 			ListType: &gexpr.Type_ListType{
// 		// 				ElemType: &gexpr.Type{
// 		// 					TypeKind: &gexpr.Type_Primitive{
// 		// 						Primitive: gexpr.Type_STRING,
// 		// 					},
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 	},
// 		// 	indigoType: indigo.List{
// 		// 		ValueType: indigo.String{},
// 		// 	},
// 		// 	wantError: false,
// 		// },
// 	}

// 	// Check if converting FROM cel TO indigo works
// 	for _, c := range cases {
// 		i, err := convertRefValToIndigo(c.celType, nil)
// 		if err != nil {
// 			t.Error(err)
// 		}

// 		if i.Type.String() != c.indigoType.String() {
// 			fmt.Println("------Mismatch types")
// 		}
// 		fmt.Println("TTTTTTT ", i.Type.String())
// 	}

// }
