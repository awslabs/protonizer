package cmd

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
)

// handle errors
func handle(e error) {
	if e != nil {

		//log error if verbose
		debug(e)

		//buld message to show user
		msg := `something went wrong 😞`
		if !verbose {
			msg += `  try -v for more info`
		}

		//show message
		errorExit(msg)
	}
}

// wraps an error with context and handles it
func handleError(context string, err error) {
	if err != nil {
		handle(fmt.Errorf("%s: %w", context, err))
	}
}

// prints message to user and exits
func errorExit(a ...interface{}) {
	fmt.Println(a...)
	os.Exit(1)
}

// if -v, log detailed message
func debug(a ...interface{}) {
	if verbose {
		log.Println(a...)
	}
}

// if -v, log detailed message
func debugFmt(format string, a ...interface{}) {
	debug(fmt.Sprintf(format, a...))
}

// SliceContains returns true if a slice contains a string
func SliceContains(s *[]string, e string, trim bool) bool {
	for _, str := range *s {
		if trim {
			str = strings.TrimSpace(str)
		}
		if str == e {
			return true
		}
	}
	return false
}

// sorts a TF module's variables
func sortTFVariables(module *tfconfig.Module) []tfconfig.Variable {
	result := []tfconfig.Variable{}
	for _, v := range module.Variables {
		result = append(result, *v)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func sortTFOutputs(module *tfconfig.Module) []tfconfig.Output {
	result := []tfconfig.Output{}
	for _, v := range module.Outputs {
		result = append(result, *v)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// recursively parses all templates in the FS with the given extension
// filepaths are used as template names to support duplicate file names
func templateParseFSRecursive(templates fs.FS, ext string, funcMap template.FuncMap) (*template.Template, error) {
	root := template.New("")
	err := fs.WalkDir(templates, "templates", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && strings.HasSuffix(path, ext) {
			if err != nil {
				return err
			}
			b, err := fs.ReadFile(templates, path)
			if err != nil {
				return err
			}
			//name the template based on the file path (excluding the root)
			parts := strings.Split(path, string(os.PathSeparator))
			name := strings.Join(parts[1:], string(os.PathSeparator))
			t := root.New(name).Funcs(funcMap)
			_, err = t.Parse(string(b))
			if err != nil {
				return err
			}
		}
		return nil
	})
	return root, err
}

// reads a template
func readTemplateFS(f string, a ...interface{}) []byte {
	result, err := fs.ReadFile(templateFS, path.Join("templates", fmt.Sprintf(f, a...)))
	handleError("reading template file", err)
	return result
}

// renders data into a template returning the result
func render(template string, data interface{}, a ...interface{}) []byte {
	var buf bytes.Buffer
	err := scaffoldTemplates.ExecuteTemplate(&buf, fmt.Sprintf(template, a...), data)
	if err != nil {
		errorExit("error executing go template:", err)
	}
	return buf.Bytes()
}
