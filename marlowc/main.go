package main

import "os"
import "io"
import "fmt"
import "flag"
import "path"
import "sync"
import "bytes"
import "strings"
import "go/build"

import "github.com/vbauerster/mpb"
import "github.com/dustin/go-humanize"

import "github.com/dadleyy/marlow/marlow"
import "github.com/dadleyy/marlow/marlow/constants"

func main() {
	cwd, e := os.Getwd()

	if e != nil {
		exit("unable to get current directory", e)
	}

	options := cliOptions{ext: constants.DefaultMarlowFileExtension}
	flag.StringVar(&options.input, "input", cwd, "the input to compile")
	flag.BoolVar(&options.stdout, "stdout", false, "print generated code to stdout")
	flag.BoolVar(&options.silent, "silent", false, "print nothing unless error")
	flag.StringVar(&options.ext, "extension", options.ext, "the file extension used for generated code")

	flag.Usage = usage
	flag.Parse()

	sourceFiles, e := loadFileNames(options.input)

	if e != nil {
		exit("unable to load package from input", e)
	}

	var progressOut io.Writer = new(bytes.Buffer)

	// If not pringing to stdout and not silent, use os.Stdout.
	if options.stdout == false && !options.silent {
		progressOut = os.Stdout
	}

	fmt.Fprintf(progressOut, "starting progress")

	total := len(sourceFiles)

	// If no files were found, exit.
	if total == 0 {
		exit("no source files found", nil)
	}

	// Create the progress bar.
	progress := mpb.New(
		mpb.WithOutput(progressOut),
		mpb.WithWaitGroup(&sync.WaitGroup{}),
	)
	bar := progress.AddBar(int64(total))

	// Keep a list of files that have been created to print out at the end.
	results := make(map[string]int64)

	for _, name := range sourceFiles {
		// Skip files that have already bee compiled.
		if strings.HasSuffix(path.Base(name), options.ext) {
			bar.IncrBy(1)
			continue
		}

		// Attempt to build the writer that we will copy the generated source into.
		writer, e := options.writerFor(name)

		if e != nil {
			exit("unable to create writer for file", e)
		}

		// Create our marlow compiler for the given file.
		reader, e := marlow.NewReaderFromFile(name)

		if e != nil {
			exit("unable to open output for file", e)
		}

		size, e := io.Copy(writer, reader)

		if e != nil {
			exit(fmt.Sprintf("unable to compile file %s", name), e)
		}

		// If no data was copied we had an no-op gen source, remove the file and continue.
		if size == 0 {
			os.Remove(options.generatedName(name))
			bar.IncrBy(1)
			continue
		}

		fmt.Fprintf(os.Stdout, "completed compilation of %s\n", name)

		results[options.generatedName(name)] = size

		// Close the destination file/buffer.
		writer.Close()

		// Let our progress bar know we're done.
		bar.IncrBy(1)
	}

	progress.Wait()

	// If no files were the target of compilation, or the silent flag was used, do nothing.
	if len(results) == 0 || options.silent == true {
		return
	}

	// As the final step, loop over all files printing out their name and size.
	fmt.Fprintln(os.Stdout, "success! files generated:")

	for name, size := range results {
		fmt.Fprintf(os.Stdout, " - %s (%s)\n", name, humanize.Bytes(uint64(size)))
	}
}

type cliOptions struct {
	input  string
	stdout bool
	silent bool
	ext    string
}

func (o *cliOptions) generatedName(input string) string {
	dir := path.Dir(input)
	name := strings.TrimSuffix(path.Base(input), path.Ext(input)) + o.ext
	return path.Join(dir, name)
}

func (o *cliOptions) writerFor(input string) (io.WriteCloser, error) {
	// If we're printing to stdout, just return a bytes.Buffer wrapped w/ a Close.
	if o.stdout == true {
		buffer := new(bytes.Buffer)
		return &closableBuffer{Buffer: buffer}, nil
	}

	full := o.generatedName(input)
	return os.Create(full)
}

// closableBuffer records are used in place of actual files when the -std flag is used.
type closableBuffer struct {
	*bytes.Buffer
}

func (b *closableBuffer) Close() error {
	fmt.Fprintf(os.Stdout, "contents:\n----\n%s----\n\n", b.String())
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
	flag.PrintDefaults()
}

func exit(msg string, e error) {
	if e != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Error: %s: %s\n", msg, e.Error()))
	} else {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Error: %s\n", msg))
	}

	flag.Usage()

	os.Exit(2)
}

// loadFileNames uses the go/build package to get a list of all valid golang files.
func loadFileNames(input string) ([]string, error) {
	stat, e := os.Stat(input)

	if e != nil {
		return nil, e
	}

	if !stat.IsDir() {
		return []string{input}, nil
	}

	pkg, e := build.Default.ImportDir(input, build.IgnoreVendor)

	if e != nil {
		return nil, e
	}

	output := make([]string, 0, len(pkg.GoFiles))

	// For each go file in the parsed package, add it's full name to the list of files.
	for _, name := range pkg.GoFiles {
		full := path.Join(input, name)
		output = append(output, full)
	}

	return output, nil
}
