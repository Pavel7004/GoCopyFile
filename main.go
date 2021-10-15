package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
)

var (
	ErrNotEnoughArgs = errors.New("Not enough arguments")
)

func copyFile(fdIn io.Reader, fdOut io.Writer, chunkSize int) (<-chan int, chan error) {
	prCh := make(chan int)
	errCh := make(chan error)

	go func() {
		last := false
		byteCount := 0
		dummyCh := make(chan struct{})
		close(dummyCh)
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
			byteCount += n
			select {
			case <-errCh:
				break
			case prCh <- byteCount:
			case <-dummyCh:
			}
		}
		close(errCh)
		close(prCh)
	}()

	return prCh, errCh
}

func parseFilenames() (string, string, error) {
	if len(os.Args) < 2 {
		return "", "", ErrNotEnoughArgs
	}
	var (
		readFile  string
		writeFile string
	)
	readFile = os.Args[1]
	if len(os.Args) >= 3 {
		writeFile = os.Args[2]
	} else {
		writeFile = "a.out"
	}
	return readFile, writeFile, nil
}

func openFiles(read_name, write_name string) (*os.File, *os.File, error) {
	fIn, err := os.Open(read_name)
	if err != nil {
		return nil, nil, err
	}
	fOut, err := os.Create(write_name)
	if err != nil {
		fIn.Close()
		return nil, nil, err
	}
	return fIn, fOut, nil
}

func main() {
	read_name, write_name, err := parseFilenames()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}
	fIn, fOut, err := openFiles(read_name, write_name)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}
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
				fmt.Printf("Error: %s\n", err.Error())
				return
			}
			fmt.Printf("Copied %s to %s.\n", read_name, write_name)
			return
		case <-exitCh:
			errCh <- nil
			return
		case n := <-prCh:
			fmt.Printf("\r%d bytes read", n)
		}
	}
}
