package cmd

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func createTarGZFile(source, target string) error {

	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		//ignore the zip itself
		if filepath.Base(path) == filepath.Base(target) {
			return nil
		}

		//ignore .git
		if strings.Contains(path, ".git") {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(path, source)
		if err != nil {
			return err
		}

		if info.IsDir() {
			header.Name += "/"
		}

		err = tw.WriteHeader(header)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}
