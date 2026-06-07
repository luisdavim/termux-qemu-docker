package utils

import (
	"io"
	"os"
)

func CopyFile(srcFile, dstFile string) error {
	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}

	defer func() { _ = out.Close() }()

	in, err := os.Open(srcFile)
	if err != nil {
		return err
	}

	defer func() { _ = in.Close() }()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Sync()
}
