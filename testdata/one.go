package testdata

import "fmt"

type Foo interface {
	expr()
}

type Bar struct{}

type Baz struct{}

func (Bar) expr() { panic("default") }
func (Baz) expr() { panic("default") }

func test(i interface{}) {
	switch i.(type) {
	case Bar:
		fmt.Println("Bar")
	case Baz:
		fmt.Println("Baz")
	}
}
