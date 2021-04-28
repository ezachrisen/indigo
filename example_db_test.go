package indigo_test

import (
	"database/sql"
	"fmt"

	"github.com/ezachrisen/indigo"
	//	_ "github.com/mattn/go-sqlite3"
)

// Example showing how to load a schema and rules from a
// database. Uncomment the driver import to run.
func Example_loadFromDatabase() {

	schemas := map[string]*indigo.Schema{}

	db, err := sql.Open("sqlite3", "./testdata/rules.db")
	if err != nil {
		fmt.Println(err)
		return
	}

	s, err := LoadSchema("student_data", db)
	if err != nil {
		fmt.Println(err)
		return
	}

	schemas[s.ID] = s

	_, err = LoadRule("old_students", schemas, db)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("OK")

}

func LoadSchema(id string, db *sql.DB) (*indigo.Schema, error) {

	rows, err := db.Query("select s.id as schema_id, s.name as schema_name, de.name as element_name, de.type as element_type from schema_elements se join schema s on s.id = se.schema_id join data_element de on de.id = se.data_element_id where s.id = '" + id + "';")

	if err != nil {
		return nil, err
	}

	s := indigo.Schema{
		ID: id,
	}

	for rows.Next() {
		var element_type string
		de := indigo.DataElement{}
		err := rows.Scan(&s.ID, &s.Name, &de.Name, &element_type)
		if err != nil {
			return nil, err
		}
		// In the database, the types are represented as strings, with the type name.
		// For example, "string", "float", etc.
		// ParseType converts from the string to the Indigo data type.
		t, err := indigo.ParseType(element_type)
		if err != nil {
			return nil, err
		}
		de.Type = t
		s.Elements = append(s.Elements, de)
	}
	return &s, nil
}

func LoadRule(id string, schemas map[string]*indigo.Schema, db *sql.DB) (*indigo.Rule, error) {

	r := indigo.Rule{
		ID: id,
	}
	var schema_id string

	qry := `SELECT expr, schema_id FROM rule WHERE id=$1;`

	row := db.QueryRow(qry, id)
	switch err := row.Scan(&r.Expr, &schema_id); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("%w, rule id: %s", err, id)
	case nil:
		s, ok := schemas[schema_id]
		if !ok {
			return nil, fmt.Errorf("unknown schema: %s", schema_id)
		}
		r.Schema = *s
	default:
		return nil, fmt.Errorf("%v, rule id: %s", err, id)
	}

	return &r, nil
}
