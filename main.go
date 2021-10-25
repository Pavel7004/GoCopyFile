package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
)

var (
	ErrNotEnoughArgs = errors.New("Not enough arguments")
)

func copyFile(ctx context.Context, fdIn io.Reader, fdOut io.Writer, chunkSize int) (<-chan int, chan error) {
	prCh := make(chan int)
	errCh := make(chan error)

	go func() {
		last := false
		byteCount := 0
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
			case <-ctx.Done():
				errCh <- ctx.Err()
				break LOOP
			case prCh <- byteCount:
			default:
			}
		}
		close(prCh)
		close(errCh)
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

func openFiles(readName, writeName string) (*os.File, *os.File, error) {
	fIn, err := os.Open(readName)
	if err != nil {
		return nil, nil, err
	}
	fOut, err := os.Create(writeName)
	if err != nil {
		fIn.Close()
		return nil, nil, err
	}
	return fIn, fOut, nil
}

func main() {
	readName, writeName, err := parseFilenames()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}
	fIn, fOut, err := openFiles(readName, writeName)
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

	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	prCh, errCh := copyFile(ctx, in, out, 4096)

	for {
		select {
		case <-exitCh:
			fmt.Printf("\rInterrupted\n")
			cancel()
		case err := <-errCh:
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				if !errors.Is(err, ctx.Err()) {
					cancel()
				}
				// Wait until errCh is closed
				<-errCh
				return
			}
			fmt.Printf("\nCopied %s to %s.\n", readName, writeName)
			cancel()
			return
		case n := <-prCh:
			fmt.Printf("\r%d bytes read", n)
		}
	}
}
