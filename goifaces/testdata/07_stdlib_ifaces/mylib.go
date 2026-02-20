package mylib

type MyError struct {
	Msg string
}

func (e MyError) Error() string {
	return e.Msg
}

type Pretty struct {
	Name string
}

func (p Pretty) String() string {
	return p.Name
}

type Bytes struct{}

func (b Bytes) Read(p []byte) (int, error) {
	return 0, nil
}
