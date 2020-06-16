package indigo

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"text/template"
)

// This file contains functions that enable users to inspect an engine's
// rules and structure. 


// StructureToTmpFile is a convenience wrapper around StructureToHTML. 
// It writes the HTML to a temporary file and returns the file name. 
func StructureToTmpFile(e Engine) (string, error) {
	html, err := StructureToHTML(e)
	if err != nil {
		return "", err
	}

	f, err := ioutil.TempFile("", "rules_*.html")
	if err != nil {
		return "", err
	}
	_, err = f.WriteString(html)
	if err != nil {
		return "", err
	}
	return f.Name(), nil

}



// StructureToHTML  walks the rule tree of an engine, printing information about
// each rule. It returns a standalone HTML page with the results. 
func StructureToHTML(e Engine) (string, error) {

	// Use this to encode .svg files to base64: https://www.base64-image.de/
	page, err := template.New("page").Funcs(template.FuncMap{
		"typeName": func(o interface{}) string {
			t := reflect.TypeOf(o)
			return t.String()
		},
		"structFieldWithName": func(name string, o interface{}, defaultValue string) string {
			r := reflect.ValueOf(o)
			v := reflect.Indirect(r)
			if v.Kind() != reflect.Struct {
				return defaultValue
			}

			f := v.FieldByName(name)
			if f.IsValid() != true {
				id := v.FieldByName("ID")
				if id.IsValid() != true {
					return defaultValue
				}
				return id.String()
			}
			return f.String()
		},
	}).Parse(`
	<!-- --------------------------- THE PAGE --!>
	<html>
	<head>

	<!-- --------------------------- CSS --!>
	<style>
	body {
		padding-left: 50px;
		padding-top: 50px;
		padding-right: 30px;
		padding-bottom: 100px;
		max-width: 700px;
	}

	.title {
		font-family: 'SF Pro Text', 'Roboto',  'Roboto', 'Arial', sans-serif;
		font-size: 20px;
		font-weight: 600;
		color: #7F7F7F;
		padding-bottom: 30px;
	}
	.ruleName {
	  font-family: 'SF Pro Text', 'Roboto',  'Roboto', 'Arial', sans-serif;
	  font-size: 14px;
	  font-weight: 500;
	  color: #7F7F7F;
	}
	.itemText {
	  font-family: 'SF Pro Text','Roboto', 'Roboto', 'Arial', sans-serif;
	  font-size: 12px;
	  color: #7F7F7F;
	}
	.ruleIcon {
	  padding-right: 5px;
	}

	ul {
		/* border:solid 1px;  */
		padding-left:2em; 
		margin:0.5em; 
		list-style: none;
		vertical-align: top;			
	}

	ul > li {
	 padding-top: 5px;
	 margin-left: -12px;
	 clear:left;
	}

	li:before {
		content:"";
		height:1em;
		width:1em;
		display:block;
		float:left;
		margin-left: -28px;
		background-repeat:no-repeat;
		content:"";
		background-size:100%;
		background-position:center;
	}

	li.rule {
		padding-top: 30px;
	}

   	li.rule:before {		  
		background-image:url('data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDI0LjEuMywgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCAxMDkuMDggMTA3LjUzIiBlbmFibGUtYmFja2dyb3VuZD0ibmV3IDAgMCAxMDkuMDggMTA3LjUzIiB4bWw6c3BhY2U9InByZXNlcnZlIj4KPGcgaWQ9IkxpZ2h0LU0iIHRyYW5zZm9ybT0ibWF0cml4KDEgMCAwIDEgMTA4OS44NSAxMTI2KSI+Cgk8cGF0aCBmaWxsPSIjMDA4QUZGIiBkPSJNLTEwMzUuMzEtMTAyNC42NWMxLjM3LDAsMi43My0wLjEsNC4xNS0wLjI0bDIuNjQsNS4wOGMwLjU0LDEuMDMsMS40MiwxLjQ2LDIuNzMsMS4zMgoJCWMxLjE3LTAuMiwxLjgxLTAuOTMsMi0yLjJsMC43OC01LjYyYzIuNjktMC42OCw1LjMyLTEuNjYsNy44MS0yLjgzbDQuMTUsMy43NmMwLjg4LDAuODMsMS44NiwwLjkzLDMuMDMsMC4zNAoJCWMxLjAzLTAuNTQsMS4zNy0xLjQ2LDEuMTItMi43M2wtMS4xMi01LjUyYzIuMjUtMS41Niw0LjM5LTMuMzIsNi4zNS01LjMybDUuMTgsMi4xNWMxLjA3LDAuNTQsMi4wNSwwLjI5LDIuOTgtMC43OAoJCWMwLjgzLTAuODgsMC45My0xLjksMC4yLTIuOTNsLTMuMDMtNC43OWMxLjYxLTIuMjUsMi45OC00LjY5LDQuMTUtNy4yM2w1LjY2LDAuMjljMS4yMiwwLjA1LDIuMS0wLjQ5LDIuNDktMS42NgoJCWMwLjM5LTEuMTcsMC4wNS0yLjEtMC44OC0yLjgzbC00LjQ0LTMuNTZjMC43My0yLjU5LDEuMjItNS4zNywxLjQyLTguMmw1LjM3LTEuNzFjMS4xNy0wLjQ0LDEuODEtMS4xNywxLjgxLTIuMzkKCQlzLTAuNjMtMi0xLjgxLTIuNDRsLTUuMzctMS42NmMtMC4yLTIuODgtMC43My01LjU3LTEuNDItOC4ybDQuNDQtMy41NmMwLjkzLTAuNjgsMS4yMi0xLjYxLDAuODgtMi43OAoJCWMtMC4zOS0xLjE3LTEuMjctMS43MS0yLjQ5LTEuNjZsLTUuNjYsMC4yNGMtMS4xNy0yLjU5LTIuNTQtNC45My00LjE1LTcuMjNsMy4wMy00Ljc5YzAuNjgtMC45OCwwLjY0LTItMC4yLTIuODgKCQljLTAuOTMtMS4wNy0xLjktMS4zMi0yLjk4LTAuNzhsLTUuMTgsMi4xYy0xLjk1LTEuOTUtNC4xLTMuNzYtNi4zNS01LjMybDEuMTItNS40N2MwLjI0LTEuMjctMC4xLTIuMi0xLjEyLTIuNzMKCQljLTEuMTctMC42My0yLjE1LTAuNTQtMy4wMywwLjM0bC00LjE1LDMuNjZjLTIuNDktMS4xMi01LjEzLTIuMDUtNy44MS0yLjc4bC0wLjc4LTUuNTdjLTAuMi0xLjI3LTAuODgtMi0yLjA1LTIuMgoJCWMtMS4yNy0wLjE1LTIuMTUsMC4yOS0yLjY5LDEuMjdsLTIuNjQsNS4wOGMtMS40Mi0wLjEtMi43OC0wLjItNC4xNS0wLjJjLTEuNDIsMC0yLjczLDAuMS00LjE1LDAuMmwtMi42OS01LjA4CgkJYy0wLjQ5LTAuOTgtMS4zNy0xLjQyLTIuNjktMS4yN2MtMS4xNywwLjItMS44NiwwLjkzLTIsMi4ybC0wLjgzLDUuNTdjLTIuNjQsMC43My01LjI3LDEuNjEtNy44MSwyLjc4bC00LjE1LTMuNjEKCQljLTAuODgtMC44OC0xLjg2LTAuOTgtMy4wMy0wLjM5Yy0wLjk4LDAuNTQtMS4zMiwxLjQ2LTEuMTIsMi43M2wxLjE3LDUuNDdjLTIuMjUsMS41Ni00LjQ0LDMuMzctNi40LDUuMzJsLTUuMTMtMi4xCgkJYy0xLjEyLTAuNTQtMi4wNS0wLjI5LTIuOTgsMC43OGMtMC44MywwLjg4LTAuODgsMS45LTAuMjQsMi44M2wzLjAzLDQuODNjLTEuNTYsMi4yOS0yLjkzLDQuNjQtNC4xLDcuMjNsLTUuNzEtMC4yNAoJCWMtMS4yMi0wLjA1LTIuMDUsMC40OS0yLjQ0LDEuNjZjLTAuNDQsMS4xNy0wLjE1LDIuMSwwLjg4LDIuNzhsNC4zOSwzLjU2Yy0wLjY4LDIuNjQtMS4xNyw1LjMyLTEuMzcsOC4xNWwtNS4zNywxLjcxCgkJYy0xLjIyLDAuNDQtMS44MSwxLjE3LTEuODEsMi40NHMwLjU5LDIsMS44MSwyLjM5bDUuMzcsMS43NmMwLjIsMi43OCwwLjY4LDUuNTcsMS4zNyw4LjE1bC00LjM5LDMuNTYKCQljLTAuOTgsMC42OC0xLjI3LDEuNjEtMC44OCwyLjgzYzAuMzksMS4xNywxLjIyLDEuNzEsMi40NCwxLjY2bDUuNzEtMC4yOWMxLjEyLDIuNTQsMi41NCw0Ljk4LDQuMSw3LjIzbC0yLjk4LDQuNzkKCQljLTAuNzMsMS4wMy0wLjY4LDIuMDUsMC4yLDIuOTNjMC45MywxLjA3LDEuODYsMS4zMiwyLjk4LDAuNzhsNS4xMy0yLjE1YzEuOTUsMiw0LjE1LDMuNzYsNi40LDUuMzJsLTEuMTcsNS41MgoJCWMtMC4yLDEuMjcsMC4xNSwyLjIsMS4xMiwyLjczYzEuMjIsMC41OSwyLjE1LDAuNDksMy4wMy0wLjM0bDQuMTUtMy43NmMyLjU0LDEuMTcsNS4xOCwyLjE1LDcuODEsMi44OGwwLjgzLDUuNTcKCQljMC4xNSwxLjI3LDAuODMsMiwyLjA1LDIuMmMxLjI3LDAuMTUsMi4xNS0wLjI5LDIuNjQtMS4zMmwyLjY5LTUuMDhDLTEwMzguMDktMTAyNC43NS0xMDM2LjcyLTEwMjQuNjUtMTAzNS4zMS0xMDI0LjY1egoJCSBNLTEwMjMuNjQtMTA3NC43NWMtMS43Ni01Ljc2LTYuMi05LjMzLTExLjcyLTkuMzNjLTEuMDMsMC0yLjEsMC4xNS0zLjU2LDAuNTlsLTE1LjIzLTI2LjEyYzUuNjItMi44OCwxMi4wMS00LjQ5LDE4Ljg1LTQuNDkKCQljMjIuNDEsMCwzOS45OSwxNy4xOSw0MS4yNiwzOS4zNkgtMTAyMy42NHogTS0xMDc2LjYyLTEwNzIuMjFjMC0xNC41NSw2Ljk4LTI3LjIsMTcuODctMzQuNjdsMTUuMjgsMjYuMTcKCQljLTIuNTksMi40NC0zLjk2LDUuNDctMy45Niw4LjY5YzAsMy4yNywxLjQyLDYuMiw0LjEsOC43NGwtMTUuNTgsMjUuNjNDLTEwNjkuNjgtMTA0NS4xNi0xMDc2LjYyLTEwNTcuODEtMTA3Ni42Mi0xMDcyLjIxegoJCSBNLTEwNDEuOS0xMDcyLjA3YzAtMy42MSwzLjA4LTYuNDksNi41OS02LjQ5YzMuNjYsMCw2LjY0LDIuODgsNi42NCw2LjQ5YzAsMy42Ni0yLjk4LDYuNTktNi42NCw2LjU5CgkJQy0xMDM4LjgyLTEwNjUuNDctMTA0MS45LTEwNjguNC0xMDQxLjktMTA3Mi4wN3ogTS0xMDM1LjMxLTEwMzAuMzdjLTYuODgsMC0xMy4zMy0xLjYxLTE4Ljk1LTQuNTRsMTUuNDgtMjUuNTQKCQljMS4zNywwLjM5LDIuNDQsMC40OSwzLjQyLDAuNDljNS42MiwwLDEwLjExLTMuNjYsMTEuNzctOS42MmgyOS41NEMtOTk1LjM3LTEwNDcuNTUtMTAxMi45NS0xMDMwLjM3LTEwMzUuMzEtMTAzMC4zN3oiLz4KPC9nPgo8L3N2Zz4K');		
    }

    li.schema:before {	
	  background-image:url('data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDI0LjEuMywgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCA5Ni43MyA4Mi41NyIgZW5hYmxlLWJhY2tncm91bmQ9Im5ldyAwIDAgOTYuNzMgODIuNTciIHhtbDpzcGFjZT0icHJlc2VydmUiPgo8ZyBpZD0iTGlnaHQtTSIgdHJhbnNmb3JtPSJtYXRyaXgoMSAwIDAgMSAxMDkyLjMyIDExMjYpIj4KCTxwYXRoIGZpbGw9IiMwMENDRTYiIGQ9Ik0tMTA4OS40NC0xMTIwLjI5aDkwLjkyYzEuNjYsMCwyLjkzLTEuMjIsMi45My0yLjgzcy0xLjI3LTIuODgtMi45My0yLjg4aC05MC45MgoJCWMtMS42NiwwLTIuODgsMS4yNy0yLjg4LDIuODhTLTEwOTEuMS0xMTIwLjI5LTEwODkuNDQtMTEyMC4yOXogTS0xMDg5LjQ0LTEwOTQuNjVoOTAuOTJjMS42NiwwLDIuOTMtMS4yNywyLjkzLTIuODgKCQljMC0xLjYxLTEuMjctMi44My0yLjkzLTIuODNoLTkwLjkyYy0xLjY2LDAtMi44OCwxLjIyLTIuODgsMi44M0MtMTA5Mi4zMi0xMDk1LjkyLTEwOTEuMS0xMDk0LjY1LTEwODkuNDQtMTA5NC42NXoKCQkgTS0xMDg5LjQ0LTEwNjkuMDdoOTAuOTJjMS42NiwwLDIuOTMtMS4yMiwyLjkzLTIuODNjMC0xLjYxLTEuMjctMi44OC0yLjkzLTIuODhoLTkwLjkyYy0xLjY2LDAtMi44OCwxLjI3LTIuODgsMi44OAoJCUMtMTA5Mi4zMi0xMDcwLjI5LTEwOTEuMS0xMDY5LjA3LTEwODkuNDQtMTA2OS4wN3ogTS0xMDg5LjQ0LTEwNDMuNDNoOTAuOTJjMS42NiwwLDIuOTMtMS4yNywyLjkzLTIuODhzLTEuMjctMi44My0yLjkzLTIuODMKCQloLTkwLjkyYy0xLjY2LDAtMi44OCwxLjIyLTIuODgsMi44M1MtMTA5MS4xLTEwNDMuNDMtMTA4OS40NC0xMDQzLjQzeiIvPgo8L2c+Cjwvc3ZnPgo=');		
	  background-size:90%;
    }

	li.expression:before {
		padding-top: 12px;
		background-image:url('data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDI0LjEuMywgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCA2Ni4zNyA2Ni42NSIgZW5hYmxlLWJhY2tncm91bmQ9Im5ldyAwIDAgNjYuMzcgNjYuNjUiIHhtbDpzcGFjZT0icHJlc2VydmUiPgo8ZyBpZD0iVGhpbi1TIiB0cmFuc2Zvcm09Im1hdHJpeCgxIDAgMCAxIDgxMy40NzggNjk2KSI+Cgk8cGF0aCBmaWxsPSIjRjQ3QTAwIiBkPSJNLTc4MC4yNy02MjkuMzVjMS4wNywwLDEuNjYtMC42OCwxLjY2LTEuOTV2LTI0LjY2YzAtOC4zLDguNjktMjEuNDgsMTkuOTItMjkuMDVsMy45Niw1LjM3CgkJYzAuOTgsMS40MiwyLjI5LDEuMTcsMi44OC0wLjU5bDQuNTktMTMuMzhjMC40OS0xLjQyLTAuMi0yLjM0LTEuNzEtMi4zNGwtMTQuMTYsMC4yNGMtMS44MSwwLjA1LTIuNDQsMS4yMi0xLjQyLDIuNTlsMy44MSw1LjIyCgkJYy0xMC43OSw3LjU3LTE4LjYsMTkuMzQtMTkuNDgsMjUuNDloLTAuMTVjLTAuODgtNi4yLTguNjktMTcuOTctMTkuNDgtMjUuNDlsMy44MS01LjEzYzEuMDMtMS4zNywwLjM5LTIuNTQtMS4zNy0yLjU5bC0xNC4xNi0wLjM5CgkJYy0xLjUxLTAuMDUtMi4yNSwwLjkzLTEuNzYsMi4zNGw0LjQ5LDEzLjQzYzAuNTQsMS43MSwxLjg2LDIsMi44OCwwLjYzbDQtNS40N2MxMS4yMyw3LjYyLDE5Ljk3LDIwLjgsMTkuOTcsMjkuMXYyNC42NgoJCUMtNzgxLjk4LTYzMC4wMy03ODEuMzktNjI5LjM1LTc4MC4yNy02MjkuMzV6Ii8+CjwvZz4KPC9zdmc+Cg==');
	}

	li.meta:before {
		background-image:url('data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDI0LjEuMywgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCA2OS4xIDc3Ljc0IiBlbmFibGUtYmFja2dyb3VuZD0ibmV3IDAgMCA2OS4xIDc3Ljc0IiB4bWw6c3BhY2U9InByZXNlcnZlIj4KPGcgaWQ9IlRoaW4tUyIgdHJhbnNmb3JtPSJtYXRyaXgoMSAwIDAgMSA4MTMuMDg3IDY5NikiPgoJPHBhdGggZmlsbD0iIzdGN0Y3RiIgZD0iTS03NTEuMTctNjU2LjA0bC0yOS4zLDI5LjNjLTcuNTIsNy41Mi0xNy42OCw2LjQ5LTI0LjEyLDBjLTYuNDktNi40NS03LjUyLTE2LjYsMC0yNC4xMmwzOC4yOC0zOC4yOAoJCWM1LjE4LTUuMTgsMTIuMDEtNC4zLDE1Ljk3LTAuMzljMy45MSwzLjkxLDQuNzksMTAuNzQtMC4zOSwxNS45N2wtMzcuNSwzNy41Yy0yLjM5LDIuMzQtNS4yNywxLjktNy4xMywwLjEKCQljLTEuODEtMS44Ni0yLjI1LTQuNzQsMC4xLTcuMTNsMjcuMzktMjcuMjljMC43My0wLjczLDAuNjMtMS43MSwwLjA1LTIuMjljLTAuNTktMC41OS0xLjQ2LTAuNjMtMi4yNSwwLjFsLTI3LjM5LDI3LjM5CgkJYy0zLjU2LDMuNjEtMy4yMiw4LjU0LTAuMjQsMTEuNTdjMy4wOCwzLjA4LDcuOTYsMy4zNywxMS41Ny0wLjI0bDM3LjU1LTM3LjU1YzYuNzQtNi43NCw1LjM3LTE1LjI4LDAuMzQtMjAuMzEKCQlzLTEzLjUzLTYuNDUtMjAuMzEsMC4zNGwtMzguMjMsMzguMjNjLTkuMTMsOS4xMy03LjYyLDIxLjE0LTAuMSwyOC43MWM3LjU3LDcuNTIsMTkuNjMsOC45OCwyOC43MS0wLjFsMjkuMjUtMjkuMjUKCQljMC42OC0wLjY4LDAuNjgtMS43NiwwLjA1LTIuMzRDLTc0OS40Ni02NTYuNzctNzUwLjQ0LTY1Ni43Mi03NTEuMTctNjU2LjA0eiIvPgo8L2c+Cjwvc3ZnPgo=');
		background-size:90%;
	}

	li.self:before{
		background-image:url('data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDI0LjEuMywgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCA3MS4wNCA2OC4wNyIgZW5hYmxlLWJhY2tncm91bmQ9Im5ldyAwIDAgNzEuMDQgNjguMDciIHhtbDpzcGFjZT0icHJlc2VydmUiPgo8cGF0aCBmaWxsPSIjQTEwMEREIiBkPSJNMCw0NS43NWMwLDEzLjM4LDkuMjgsMjIuMzEsMjMuOTcsMjIuMzFoMTIuMjZjMS4wMywwLDEuNzYtMC43MywxLjc2LTEuNzFzLTAuNjgtMS43MS0xLjc2LTEuNzFIMjQuMDIKCWMtMTIuNywwLTIwLjU2LTcuNzYtMjAuNTYtMTkuMDRjMC0xMS4yOCw3Ljg2LTE4LjksMjAuNTYtMTguOWgzMy4zNWw4LjQ1LTAuMmwtNy45MSw3LjQ3TDQ0Ljc4LDQ3LjAyCgljLTAuMjksMC4yOS0wLjQ5LDAuNzMtMC40OSwxLjI3YzAsMC45OCwwLjczLDEuNzEsMS43NiwxLjcxYzAuNDksMCwwLjkzLTAuMiwxLjI3LTAuNTRsMjMuMTktMjMuMTljMC4zNC0wLjM0LDAuNTQtMC43OCwwLjU0LTEuMjcKCXMtMC4yLTAuOTMtMC41NC0xLjI3TDQ3LjMxLDAuNTRDNDYuOTcsMC4yLDQ2LjUzLDAsNDYuMDQsMGMtMS4wMywwLTEuNzYsMC43OC0xLjc2LDEuNzFjMCwwLjU0LDAuMiwwLjk4LDAuNDksMS4yN2wxMy4xMywxMy4wOQoJbDcuOTEsNy40MmwtOC40NS0wLjJoLTMzLjJDOS4zMywyMy4yOSwwLDMyLjMyLDAsNDUuNzV6Ii8+Cjwvc3ZnPgo=');
	}
	</style>
	</head>

	<body>

	<!-- --------------------------- TEMPLATE FOR A RULE --!>
	{{define "ruleSection"}}
		<li class="rule">			
			<span class="ruleName">{{.ID}}</span>
			<ul>
				{{if .Expr }}
					<li class="expression"><span class="itemText">{{.Expr}}</span></li>	
				{{end}}
				{{if .Schema.ID }}
					<li class="schema"><span class="itemText">{{.Schema.ID}}</span></li>
				{{end}}
				{{if .Self }}
					<li class="self"><span class="itemText">{{structFieldWithName "Name" .Self "(no name)"}} ({{typeName .Self}})</span></li>
				{{end}}
				{{if .Meta }}
					<li class="meta"><span class="itemText">{{structFieldWithName "Name" .Meta "(no name)"}} ({{typeName .Meta}})</span></li>
				{{end}}
				{{if .Rules }}	
					{{range $key, $value := .Rules }}
						{{template "ruleSection" $value}}
					{{end}}
				
				{{end}}		
			</ul>
		</li>
	{{end}}

	<span class="title">Rule Tree</span>


	<!-- --------------------------- GENERATE THE RULE TREE--!>
	{{if . }}
		{{range $key, $value := . }}
			<ul>
				{{template "ruleSection" $value}}
			</ul>
		{{end}}
	{{else}}
		There are no rules defined
	{{end}}
	</body>
	</html>
   `)

	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = page.ExecuteTemplate(buf, "page", e.Rules())
	if err != nil {
		return "", err
	}
	return buf.String(), nil 
}

