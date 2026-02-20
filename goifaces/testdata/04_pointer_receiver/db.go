package db

type Closer interface {
	Close() error
}

type Connection struct{}

func (c *Connection) Close() error {
	return nil
}
