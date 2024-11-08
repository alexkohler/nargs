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
