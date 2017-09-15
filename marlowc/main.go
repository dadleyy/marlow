package main

import "os"
import "io"
import "fmt"
import "flag"
import "time"
import "path"
import "sync"
import "bytes"
import "go/build"
import "github.com/vbauerster/mpb"
import "github.com/vbauerster/mpb/cwriter"
import "github.com/vbauerster/mpb/decor"
import "github.com/dadleyy/marlow/marlow"

type closableBuffer struct {
	*bytes.Buffer
}

func (b *closableBuffer) Close() error {
	fmt.Fprintf(os.Stdout, "contents:\n----\n%s----\n\n", b.String())
	return nil
}

func buildOuput(filename string) (io.WriteCloser, error) {
	return &closableBuffer{
		Buffer: new(bytes.Buffer),
	}, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func exit(msg string, e error) {
	if e != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s: %s\n", msg, e.Error()))
	} else {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", msg))
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
	}{}

	flag.StringVar(&options.directory, "directory", cwd, "the directory to compile")

	flag.Usage = usage
	flag.Parse()

	if s, e := os.Stat(options.directory); e != nil || s.IsDir() == false {
		exit("must provide a valid directory for compilation", nil)
	}

	pkg, err := build.Default.ImportDir(options.directory, 0)

	if err != nil {
		exit("unable to load package from directory", err)
	}

	p := mpb.New(
		mpb.Output(cwriter.New(bytes.NewBuffer([]byte{}))),
	)

	total := 30
	name := fmt.Sprintf("compiling: %s", options.directory)

	bar := p.AddBar(int64(total),
		mpb.PrependDecorators(decor.StaticName(name, len(name), 0)),
		mpb.AppendDecorators(decor.Percentage(5, 0)),
	)

	wg := sync.WaitGroup{}

	for _, name := range pkg.GoFiles {
		wg.Add(1)
		fullName := path.Join(options.directory, name)

		buffer, e := buildOuput(fullName)

		if e != nil {
			exit("unable to open target file", e)
		}

		output := marlow.NewWriter(buffer)

		reader, e := os.Open(fullName)

		if e != nil {
			exit("unable to open output for file", e)
		}

		if _, e := io.Copy(output, reader); e != nil {
			panic(e)
			exit(fmt.Sprintf("unable to compile file %s", fullName), e)
		}

		time.Sleep(time.Second / 4)
		bar.Incr(1)

		reader.Close()
		output.Close()
		buffer.Close()
		wg.Done()
	}

	wg.Wait()

	p.Stop()
}
