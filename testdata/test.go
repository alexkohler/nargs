package main

import "fmt"

// Unused function parameter on function
func funcOne(a int, b int, c int) int {
	return a + b
}

// Unused function parameter on method with receiver
type f struct{}

func (f) funcTwo(a int, b int, c int) int {
	return a + b
}

// Unused function receiver. Unused receivers are NOT flagged by default. Flagging unused function receivers
// can be enabled by setting the -receivers flag to true.
func (recv f) funcThree() int {
	return 5
}

// Unused named returns. Unused named returns are NOT flagged by deault. Flagging unused named returns
// can be enabled by setting the -named_returns flag to true.
func funcFour() (namedReturn int) {
	return
}

// Unused closure parameters inside function
func unusedClosureParamInsideFunction() {
	closureOne := func(v int) {
		enclosed := 2
		enclosed++
	}
	c(1)
}

// Unused function as parameter
func unusedFunc(f func()) {
}

// Unused closure parameter in package scoped closure
var closureTwo = func(i int) {
	fmt.Println()
}

func variableCapturedByClosure(r int) {
	// note that both r and n ARE used here
	feedTokens := func(n int) error {
		n = r
		return nil
	}
	feedTokens(5)
}

//TODO - functions as keys
// var usedAsGlobalInterfaceMapValue = map[string]interface{}{
// 	"someFunc": func(i int, s string) {
// 		if i == 0 {
// 			println()
// 		}
// 	},
// }
