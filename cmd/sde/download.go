package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// WriteCounter counts the number of bytes written to a stream.
type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

// PrintProgress prints the download progress.
func (wc WriteCounter) PrintProgress() {
	// Clear the line and print the progress.
	fmt.Printf("Downloading... %d MB complete", wc.Total/1024/1024)
}

// downloadFile downloads a URL to a file. It will overwrite the file if it already exists.
func downloadFile(filepath string, url string) error {
	// Create the file with a temporary name
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		out.Close()
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create our progress counter and tee the response body to it
	counter := &WriteCounter{}
	reader := io.TeeReader(resp.Body, counter)

	// Write the body to file
	_, err = io.Copy(out, reader)
	if err != nil {
		out.Close()
		return err
	}

	fmt.Println() // New line after download completes
	out.Close()

	// Rename the temp file to the final name
	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}

	return nil
}
