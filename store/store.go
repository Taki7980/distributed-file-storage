package store

import (
	"io"
	"log"
	"os"
	"strings"
)

type PathTransformFunc func(string) string

type StoreOpt struct {
	PathTransformFunc PathTransformFunc
}

var DefaultPathTransformFunc = func(key string) string {
	return key
}

func sanitizeFilename(filename string) string {
	replacer := strings.NewReplacer(
		":", "_",
		"[", "",
		"]", "",
		"/", "_",
		"\\", "_",
	)
	return replacer.Replace(filename)
}

type Store struct {
	StoreOpt
}

func NewStore(opts StoreOpt) *Store {
	return &Store{
		StoreOpt: opts,
	}
}

func (s *Store) WriteStream(key string, r io.Reader) error {
	if err := os.MkdirAll("message_storage", os.ModePerm); err != nil {
		return err
	}

	pathName := s.PathTransformFunc(key)
	pathName = sanitizeFilename(pathName)
	pathAndFilename := "message_storage/" + pathName + ".dat"

	f, err := os.Create(pathAndFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	log.Printf("written (%d) bytes to disk: %s", n, pathAndFilename)

	return nil
}
