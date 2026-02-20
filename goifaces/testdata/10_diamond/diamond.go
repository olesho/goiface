package diamond

type Saver interface {
	Save() error
}

type Loader interface {
	Load() error
}

type Persister interface {
	Save() error
	Load() error
}

type DB struct{}

func (d DB) Save() error {
	return nil
}

func (d DB) Load() error {
	return nil
}
