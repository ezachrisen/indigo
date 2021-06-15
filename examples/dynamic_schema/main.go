package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Example showing how to load a schema and rules from a
// database.
//
// Rather than import the the proto messages at compile time,
// the types are loaded dynamically at runtime from a descriptor set
// (school.descriptor).
func main() {

	schemas := map[string]*indigo.Schema{}

	err := loadProtoDescriptors("school.descriptor")
	if err != nil {
		fmt.Println("Error loading proto descriptors:", err)
		return
	}

	fmt.Println("Loaded proto descriptors")

	db, err := sql.Open("sqlite3", "rules.db")
	if err != nil {
		fmt.Println(err)
		return
	}

	s, err := loadSchema("student_data", db)
	if err != nil {
		fmt.Println("loading student_data schema: ", err)
		return
	}

	schemas[s.ID] = s
	fmt.Println("Loaded schema")

	r, err := loadRule("old_students", schemas, db)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Loaded rules")

	e := indigo.NewEngine(cel.NewEvaluator())
	err = e.Compile(r)
	if err != nil {
		fmt.Printf("compling rule '%s': %v", r.ID, err)
		return
	}
	fmt.Println("Compiled rules")
	fmt.Println("Schema and Rules:")
	fmt.Println(s)
	fmt.Println(r)

}

func loadProtoDescriptors(fname string) error {

	protoFile, err := ioutil.ReadFile(fname)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	set := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(protoFile, set); err != nil {
		return fmt.Errorf("unmarshaling proto file: %w", err)
	}

	fds := set.GetFile()
	for _, fdproto := range fds {

		// Initialize the File descriptor object
		fd, err := protodesc.NewFile(fdproto, protoregistry.GlobalFiles)
		if err != nil {
			return fmt.Errorf("creating new file descriptor: %w", err)
		}

		existing, _ := protoregistry.GlobalFiles.FindFileByPath(fd.Path())
		if existing != nil {
			continue
		}

		err = protoregistry.GlobalFiles.RegisterFile(fd)
		if err != nil {
			return fmt.Errorf("registering file descriptor: %w", err)
		}

		// Register the messages
		mds := fd.Messages()
		for i := 0; i < mds.Len(); i++ {
			m := mds.Get(i)
			msg := dynamicpb.NewMessage(m)
			if msg == nil {
				return fmt.Errorf("getting dynamic message: %w", err)
			}
			err := protoregistry.GlobalTypes.RegisterMessage(msg.ProtoReflect().Type())
			if err != nil {
				return fmt.Errorf("registering message: %w", err)
			}
		}

	}

	// Check that we can get one of the messages in our descriptor file
	_, err = protoregistry.GlobalTypes.FindMessageByName("testdata.school.StudentSummary")
	if err != nil {
		return fmt.Errorf("finding message by name: %w", err)
	}

	return nil
}

func loadSchema(id string, db *sql.DB) (*indigo.Schema, error) {

	rows, err := db.Query("select s.id as schema_id, s.name as schema_name, de.name as element_name, de.type as element_type from schema_elements se join schema s on s.id = se.schema_id join data_element de on de.id = se.data_element_id where s.id = '" + id + "';")

	if err != nil {
		return nil, err
	}

	s := indigo.Schema{
		ID: id,
	}

	for rows.Next() {
		var typeAsString string
		de := indigo.DataElement{}
		err := rows.Scan(&s.ID, &s.Name, &de.Name, &typeAsString)
		if err != nil {
			return nil, err
		}

		de.Type, err = indigo.ParseType(typeAsString)
		if err != nil {
			return nil, err
		}
		s.Elements = append(s.Elements, de)
	}
	return &s, nil
}

func loadRule(id string, schemas map[string]*indigo.Schema, db *sql.DB) (*indigo.Rule, error) {

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
