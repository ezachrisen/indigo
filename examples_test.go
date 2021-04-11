package indigo_test

// // This example demonstrates applying a closure to
// // all rules in a hierarchy
// func ExampleApplyFunc() {

// 	r := indigo.Rule{
// 		ID: "A",
// 		Rules: map[string]*indigo.Rule{
// 			"B": &indigo.Rule{ID: "B"},
// 			"C": &indigo.Rule{
// 				ID: "C",
// 				Rules: map[string]*indigo.Rule{
// 					"c1": &indigo.Rule{ID: "c1"},
// 				},
// 			},
// 		},
// 	}

// 	ids := []string{}

// 	// The closure will collect the IDs of rules visited
// 	err := r.Apply(func(r *indigo.Rule) error {
// 		ids = append(ids, r.ID)
// 		return nil
// 	})
// 	if err != nil {
// 		fmt.Println("Error: ", err)
// 		return
// 	}
// 	fmt.Printf("%+v\n", ids)
// 	// Unordered output: [A B C c1]
// }
