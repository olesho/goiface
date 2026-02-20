package io2

type Reader interface {
	Read(p []byte) (int, error)
}

type Closer interface {
	Close() error
}

type ReadCloser interface {
	Reader
	Closer
}

type MyFile struct{}

func (f MyFile) Read(p []byte) (int, error) {
	return 0, nil
}

func (f MyFile) Close() error {
	return nil
}
