// Note: a lot of this code was copied from PaceDev's OTO framework.

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

func main() {
	if err := run(os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(stdout io.Writer, args []string) error {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Println(args[0] + ` usage:
	gentype -types=type,type2,type3...`)
	}

	var (
		types = flags.String("types", "", "")
		v     = flags.Bool("v", false, "produce verbose  output")
	)
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	// template := "struct.go.plush"

	// if template == "" {
	// 	return errors.New("missing template")
	// }

	if types == nil || len(*types) == 0 {
		return errors.New("missing types")
	}

	parser := newParser(flags.Args()...)

	parser.Verbose = *v

	for _, v := range strings.Split(*types, ",") {
		parser.IncludeTypes[v] = nil
	}

	if parser.Verbose {
		log.Println("gentype")
		log.Println("Included types: ", *types)
		if os.Getenv("GOPACKAGE") != "" {
			log.Printf("go generate called from %s:%s\n", os.Getenv("GOFILE"), os.Getenv("GOLINE"))
		}
		path, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}
		log.Println("Path: ", path)
	}

	def, err := parser.parse()
	if err != nil {
		return err
	}

	if len(def.Objects) == 0 {
		if parser.Verbose {
			log.Println("No matching structs found; quitting")
		}
		return nil
	}

	// b, err := ioutil.ReadFile(template)
	// if err != nil {
	// 	return err
	// }
	out, err := render(def)
	if err != nil {
		return err
	}

	outfile := "gentype_generated.cel.go"
	var w io.Writer = stdout
	if outfile != "" {
		f, err := os.Create(outfile)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	if _, err := io.WriteString(w, out); err != nil {
		return err
	}
	if parser.Verbose {
		var methodsCount int
		for i := range def.Services {
			methodsCount += len(def.Services[i].Methods)
		}
		fmt.Println()
		fmt.Printf("\tTotal Structs: %d\n", len(def.Objects))
		fmt.Printf("\tOutput size: %s\n", humanize.Bytes(uint64(len(out))))
	}
	return nil
}

// parseParams returns a map of data parsed from the params string.
func parseParams(s string) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	if s == "" {
		// empty map for an empty string
		return params, nil
	}
	pairs := strings.Split(s, ",")
	for i := range pairs {
		pair := strings.TrimSpace(pairs[i])
		segs := strings.Split(pair, ":")
		if len(segs) != 2 {
			return nil, errors.New("malformed params")
		}
		params[strings.TrimSpace(segs[0])] = strings.TrimSpace(segs[1])
	}
	return params, nil
}
