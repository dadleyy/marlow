package marlow

import "io"
import "os"
import "fmt"
import "sync"
import "bytes"
import "strings"
import "path/filepath"
import "go/token"
import "go/parser"
import "go/format"

import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

// Compile is responsible for reading from a source and writing the generated marlow code into a destination.
func Compile(destination io.Writer, reader io.Reader) error {
	fs := token.NewFileSet()
	packageAst, e := parser.ParseFile(fs, "", reader, parser.AllErrors|parser.ParseComments)

	if e != nil {
		return e
	}

	// Check to see if we are ignoring this source via the comments.
	for _, c := range packageAst.Comments {
		ignored := strings.Contains(c.Text(), constants.IgnoreSourceDirective)

		if ignored {
			return nil
		}
	}

	buffered, packageName := new(bytes.Buffer), packageAst.Name.String()
	importChannel, recordReaders := make(chan string), make([]io.Reader, 0, len(packageAst.Decls))

	// Establish a list of all the source package imports - we will use this to determine the full import name from an
	// import sent by our features if all the feature can determine is the local name of the import.
	packageImports := make(map[string]string)

	for _, i := range packageAst.Imports {
		cleansed := strings.Trim(i.Path.Value, "\"")
		local := filepath.Base(cleansed)

		// Save a reference to the full import path from the package name (final part of the import path).
		packageImports[local] = cleansed
	}

	// Iterate over the declarations and construct the record store from the loaded ast.
	for _, d := range packageAst.Decls {
		reader, ok := newRecordReader(d, importChannel)

		// Only deal with struct type declarations.
		if !ok {
			continue
		}

		recordReaders = append(recordReaders, reader)
	}

	// If no marlow records were found, just abort.
	if len(recordReaders) == 0 {
		return nil
	}

	// Write out the main package information
	packageWriter := writing.NewGoWriter(buffered)
	packageWriter.Comment(constants.CompilerHeader)
	packageWriter.Println("")
	packageWriter.WritePackage(packageName)

	wg := &sync.WaitGroup{}

	wg.Add(1)

	// In a separate goroutine, iterate over all the received import names, adding them to the buffered output.
	go func() {
		importList := make(map[string]bool)

		for importName := range importChannel {
			// Check to see if the import sent to us is actually referring to one of the package imports from the source file.
			fullImportPath, isLocal := packageImports[importName]

			// If we are referring to an import from the source package, use it's full name.
			if isLocal {
				importName = fullImportPath
			}

			if _, dupe := importList[importName]; dupe == true {
				continue
			}

			importList[importName] = true
			packageWriter.WriteImport(importName)
		}

		wg.Done()
	}()

	// Prepare the intermediate buffer that will be used to write each of the generated api.
	records := new(bytes.Buffer)

	// Copy all of the table readers into our temporary table buffer.
	if _, e := io.Copy(records, io.MultiReader(recordReaders...)); e != nil {
		return e
	}

	// Close the import writer, and wait for it to finish (was writing imports during record copying).
	close(importChannel)
	wg.Wait()

	// Copy the generated table constructs into the final buffer.
	if _, e := io.Copy(buffered, records); e != nil {
		return e
	}

	// Use the go/format package to format the generated code, ensuring it is valid golang code.
	formatted, e := format.Source(buffered.Bytes())

	// If an error was thrown at this point, its typically a prolem with the marlow generator itself.
	if e != nil {
		return fmt.Errorf("%s (error %v) source:\n%s", constants.InvalidGeneratedCodeError, e, buffered)
	}

	_, e = io.Copy(destination, bytes.NewBuffer(formatted))

	return e
}

// NewReaderFromFile opens the requested filename and returns an io.Reader that represents the compiled source.
func NewReaderFromFile(filename string) (io.Reader, error) {
	source, e := os.Open(filename)

	if e != nil {
		return nil, e
	}

	pr, pw := io.Pipe()

	go func() {
		defer source.Close()
		e := Compile(pw, source)
		pw.CloseWithError(e)
	}()

	return pr, nil
}
