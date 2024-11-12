package main

import "fmt"

func main() {
	fmt.Println("Hello world!")
}

func FuncVars(x int) {
	do := func() {
		fmt.Println(x)
	}
	do()
}

func newTypePair[K any, V any]() {
}

func testIndexListExpr() {
	newTypePair[int, int]
}
