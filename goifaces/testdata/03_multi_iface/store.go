package store

type Reader interface {
	Read(id string) ([]byte, error)
}

type Writer interface {
	Write(id string, data []byte) error
}

type ReadWriter interface {
	Reader
	Writer
}

type MemStore struct{}

func (m MemStore) Read(id string) ([]byte, error) {
	return nil, nil
}

func (m MemStore) Write(id string, data []byte) error {
	return nil
}

type ReadOnlyCache struct{}

func (r ReadOnlyCache) Read(id string) ([]byte, error) {
	return nil, nil
}
