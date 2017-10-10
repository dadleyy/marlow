package main

import "os"
import "io"
import "fmt"
import "flag"
import "time"
import "path"
import "sync"
import "bytes"
import "strings"
import "go/build"
import "github.com/vbauerster/mpb"
import "github.com/vbauerster/mpb/decor"
import "github.com/dadleyy/marlow/marlow"

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

	for _, name := range pkg.GoFiles {
		full := path.Join(input, name)
		output = append(output, full)
	}

	return output, nil
}

func main() {
	cwd, err := os.Getwd()

	if err != nil {
	}

	options := struct {
		input  string
		stdout bool
		silent bool
	}{}

	flag.StringVar(&options.input, "input", cwd, "the input to compile")
	flag.BoolVar(&options.stdout, "stdout", false, "print generated code to stdout")
	flag.BoolVar(&options.silent, "silent", false, "print nothing unless error")

	flag.Usage = usage
	flag.Parse()

	if _, e := os.Stat(options.input); e != nil {
		exit("must provide a valid input for compilation", nil)
	}

	sourceFiles, err := loadFileNames(options.input)

	if err != nil {
		exit("unable to load package from input", err)
	}

	var progressOut io.Writer = new(bytes.Buffer)

	if options.stdout == false && !options.silent {
		progressOut = os.Stdout
	}

	total, name, done := len(sourceFiles), fmt.Sprintf("compiling files"), make(chan struct{})

	progress := mpb.New(
		mpb.Output(progressOut),
		mpb.WithWidth(100),
		mpb.WithRefreshRate(time.Millisecond),
		mpb.WithShutdownNotifier(done),
	)

	bar := progress.AddBar(int64(total),
		mpb.PrependDecorators(
			decor.StaticName(name, len(name), 0),
			decor.ETA(4, decor.DSyncSpace),
		),
		mpb.AppendDecorators(decor.Percentage(5, 0)),
	)

	wg := sync.WaitGroup{}

	if len(sourceFiles) == 0 {
		exit("no source files found", nil)
	}

	writtenFiles := make([]string, 0, len(sourceFiles))

	for _, name := range sourceFiles {
		sourceDir := path.Dir(name)

		if strings.HasSuffix(path.Base(name), ".marlow.go") {
			continue
		}

		wg.Add(1)

		var buffer io.WriteCloser = &closableBuffer{Buffer: new(bytes.Buffer)}

		if options.stdout == false {
			var e error

			destName := path.Join(
				sourceDir,
				fmt.Sprintf("%s.marlow.go", strings.TrimSuffix(path.Base(name), path.Ext(name))),
			)

			if e := os.Remove(destName); e != nil && os.IsNotExist(e) == false {
				exit("unable to remove file", e)
			}

			buffer, e = os.Create(destName)

			if e != nil {
				exit("unable to write file", e)
			}

			writtenFiles = append(writtenFiles, destName)
		}

		reader, e := marlow.NewReaderFromFile(name)

		if e != nil {
			exit("unable to open output for file", e)
		}

		if _, e := io.Copy(buffer, reader); e != nil {
			exit(fmt.Sprintf("unable to compile file %s", name), e)
		}

		buffer.Close()
		bar.Incr(1)
		wg.Done()
	}

	wg.Wait()
	bar.Complete()
	progress.Stop()
	<-done

	if (len(writtenFiles) >= 1) != true {
		return
	}

	if options.silent == true {
		return
	}

	fmt.Fprintln(os.Stdout, "success! files generated:")

	for _, fn := range writtenFiles {
		fmt.Fprintf(os.Stdout, " - %s\n", fn)
	}
}
