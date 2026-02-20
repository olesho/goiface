package animals

type Speaker interface {
	Speak() string
}

type Dog struct{}

func (d Dog) Speak() string { return "woof" }

type Cat struct{}

func (c Cat) Speak() string { return "meow" }

type Fish struct{} // no Speak — should NOT appear in diagram
