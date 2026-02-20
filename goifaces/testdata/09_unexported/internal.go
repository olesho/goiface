package internal

type walker interface {
	walk()
}

type Runner interface {
	Run()
}

type dog struct{}

func (d dog) walk() {}
func (d dog) Run()  {}

type Cat struct{}

func (c Cat) Run() {}
