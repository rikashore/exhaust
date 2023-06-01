package testdata

import "fmt"

type Foo interface {
	expr()
}

type Bar struct{}

type Baz struct{}

type Quux struct{}

func (Bar) expr()  { panic("default") }
func (Baz) expr()  { panic("default") }
func (Quux) expr() { panic("default") }

func test(i Foo) {
	switch i.(type) {
	case Bar:
		fmt.Println("Bar")
	case Baz:
		fmt.Println("Baz")
	}
}
