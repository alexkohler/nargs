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

// Unused function receiver. Unused receivers are flagged by default. Flagging unused function receivers
// can be disabled by setting the -receivers flag to false.
func (recv f) funcThree() int {
	return 5
}

// Unused named returns. Unused named returns are NOT flagged by deault. Flagging unused named returns
// can be enabled by setting the -named_returns flag to true.
func funcFour() (namedReturn int) {
	return
}

func closure() {
	c := func(v int) {
		enclosed := 2
		enclosed++
	}
	c(1)
}

func unusedFunc(f func()) {
}
