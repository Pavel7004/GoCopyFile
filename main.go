package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
)

func copyFile(fdIn io.Reader, fdOut io.Writer, chunkSize int) (chan int, chan error) {
	prCh := make(chan int)
	errCh := make(chan error)
	go func() {
		last := false
		for !last {
			chunk := make([]byte, chunkSize)
			n, err := fdIn.Read(chunk)
			if err != nil && !errors.Is(err, io.EOF) {
				errCh <- err
				break
			}
			if errors.Is(err, io.EOF) {
				last = true
			}
			n, err = fdOut.Write(chunk)
			if err != nil {
				errCh <- err
				break
			}
			prCh <- n
		}
		close(errCh)
		close(prCh)
	}()
	return prCh, errCh
}

func parse_filenames() (read_file string, write_file string) {
	if len(os.Args) < 2 {
		fmt.Println("Not enough arguments")
		return
	}
	read_file = os.Args[1]
	if len(os.Args) >= 3 {
		write_file = os.Args[2]
	} else {
		write_file = "a.out"
	}
	return
}

func open_files(read_name, write_name string) (fIn *os.File, fOut *os.File) {
	fIn, err := os.Open(read_name)
	if err != nil {
		panic("Open in err")
	}
	fOut, err = os.Create(write_name)
	if err != nil {
		panic("Open out err")
	}
	return
}

func main() {
	read_name, write_name := parse_filenames()
	fIn, fOut := open_files(read_name, write_name)
	defer func() {
		fIn.Close()
		fOut.Close()
	}()
	in := bufio.NewReader(fIn)
	out := bufio.NewWriter(fOut)

	exitCh := make(chan os.Signal)
	signal.Notify(exitCh, os.Interrupt)

	prCh, errCh := copyFile(in, out, 4096)

	for {
		select {
		case err := <-errCh:
			if err != nil {
				panic(err)
			}
			fmt.Println()
			return
		case <-exitCh:
			close(exitCh)
			close(prCh)
			close(errCh)
			return
		case n := <-prCh:
			fmt.Printf("\r%d bytes read", n)
		}
	}
}
