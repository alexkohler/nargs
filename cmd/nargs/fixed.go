// main.go
package main

// Unused function parameter on function
func funcOne(a int, b int, c int) int {
	return a + b
}

// Unused function parameter on method with receiver
type f struct{}

func (f) funcTwo(a int, b int, c int) int {
	return a + b
}

// Unused function receiver and unused function parameter
func (recv f) funcThree(z int) int {
	return 5
}

// Unused named returns
func funcFour() (namedReturn int) {

	return
}
