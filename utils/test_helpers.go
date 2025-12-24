package utils

import (
	"bytes"
	"io"
	"os"
)

// CaptureStdout captures the output written to os.Stdout during the execution of f.
func CaptureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
