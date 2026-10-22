package store

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const defaultRootFolderName = "storage"

type PathKey struct {
	PathName string
	Filename string
}

func (p PathKey) FullPath() string {
	return filepath.Join(p.PathName, p.Filename)
}

func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, string(filepath.Separator))
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

type PathTransformFunc func(string) PathKey

var CASPathTransformFunc = func(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashStr) / blockSize
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		PathName: filepath.Join(paths...),
		Filename: hashStr,
	}
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		Filename: key,
	}
}

type StoreOpt struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpt
}

func NewStore(opts StoreOpt) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = CASPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}
	return &Store{
		StoreOpt: opts,
	}
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := filepath.Join(s.Root, pathKey.FullPath())

	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := filepath.Join(s.Root, pathKey.FullPath())

	if err := os.Remove(fullPathWithRoot); err != nil {
		return err
	}

	return s.pruneEmptyParents(pathKey)
}

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

func (s *Store) WriteStream(key string, r io.Reader) error {
	_, err := s.writeStream(key, r)
	return err
}

func (s *Store) Read(key string) (io.ReadCloser, error) {
	return s.readStream(key)
}

// Private functions below (me.txt convention: public top, private bottom)

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := filepath.Join(s.Root, pathKey.FullPath())
	return os.Open(fullPathWithRoot)
}

func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := filepath.Join(s.Root, pathKey.PathName)

	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return 0, err
	}

	fullPathWithRoot := filepath.Join(s.Root, pathKey.FullPath())

	f, err := os.Create(fullPathWithRoot)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (s *Store) pruneEmptyParents(pathKey PathKey) error {
	parent := filepath.Join(s.Root, pathKey.PathName)
	cleanRoot := filepath.Clean(s.Root)

	for parent != cleanRoot && strings.HasPrefix(parent, cleanRoot) {
		err := os.Remove(parent)
		if err != nil {
			break
		}
		parent = filepath.Dir(parent)
	}
	return nil
}

