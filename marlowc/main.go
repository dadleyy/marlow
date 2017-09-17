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

func main() {
	cwd, err := os.Getwd()

	if err != nil {
	}

	options := struct {
		directory string
		stdout    bool
	}{}

	flag.StringVar(&options.directory, "directory", cwd, "the directory to compile")
	flag.BoolVar(&options.stdout, "stdout", false, "print generated code to stdout")

	flag.Usage = usage
	flag.Parse()

	if s, e := os.Stat(options.directory); e != nil || s.IsDir() == false {
		exit("must provide a valid directory for compilation", nil)
	}

	pkg, err := build.Default.ImportDir(options.directory, 0)

	if err != nil {
		exit("unable to load package from directory", err)
	}

	var progressOut io.Writer = new(bytes.Buffer)

	if options.stdout == false {
		progressOut = os.Stdout
	}

	total, name, done := len(pkg.GoFiles), fmt.Sprintf("compiling: %s", pkg.Name), make(chan struct{})

	progress := mpb.New(
		mpb.Output(progressOut),
		mpb.WithWidth(100),
		mpb.WithRefreshRate(10*time.Millisecond),
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

	for _, name := range pkg.GoFiles {
		fullName := path.Join(options.directory, name)

		if strings.HasSuffix(path.Base(fullName), ".marlow.go") {
			continue
		}

		wg.Add(1)

		var buffer io.WriteCloser = &closableBuffer{Buffer: new(bytes.Buffer)}

		if options.stdout == false {
			var e error

			destName := path.Join(
				options.directory,
				fmt.Sprintf("%s.marlow.go", strings.TrimSuffix(path.Base(fullName), path.Ext(fullName))),
			)

			if e := os.Remove(destName); e != nil && os.IsNotExist(e) == false {
				exit("unable to remove file", e)
			}

			buffer, e = os.Create(destName)

			if e != nil {
				exit("unable to write file", e)
			}
		}

		reader, e := marlow.NewReaderFromFile(fullName)

		if e != nil {
			exit("unable to open output for file", e)
		}

		if _, e := io.Copy(buffer, reader); e != nil {
			exit(fmt.Sprintf("unable to compile file %s", fullName), e)
		}

		buffer.Close()
		bar.Incr(1)
		wg.Done()
	}

	wg.Wait()
	bar.Complete()
	progress.Stop()
	<-done
}
