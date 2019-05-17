# nargs

nargs is a Go static analysis tool to find unused arguments in function declarations.

## Installation

    go get -u github.com/alexkohler/nargs/cmd/nargs	

## Usage

Similar to other Go static anaylsis tools (such as golint, go vet) , nakedret can be invoked with one or more filenames, directories, or packages named by its import path. Nakedret also supports the `...` wildcard. 

    nargs files/directories/packages

## Purpose

Often, parameters will be added to functions (such as a constructor), and then not actually used within the function. This tools was written to find these types of functions.

## Example

Some simple examples
```Go
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

//TODO - nargs isn't picking this up
func nakedret() (l int) {


	return
}

```

```Bash
./nargs main.go 
main.go:5 funcOne found unused parameter c
main.go:12 funcTwo found unused parameter c
main.go:17 funcThree found unused parameter recv
main.go:17 funcThree found unused parameter z
```

## FAQ

### How should these issues be fixed?

nargs ignores function variables using the blank identifier `_`, and encourages the use of the blank identifier in the event that the parameter cannot be removed from the function due to implementing an interface or function typedef. If this is the case, the following can be done to fix the above example:

```Go
// fixed.go


```

### How is this different than [unparam](https://github.com/mvdan/unparam)?

`unparam` errs on the safe side to minimize false positives (ignoring functions that satisfy an interface, etc.). `nargs` takes a more aggressive approach and encourages the use of the blank identifier `_` for function parameters that are intentionally not used. unparam operates using the [ssa](https://godoc.org/golang.org/x/tools/go/ssa) package, whereas nargs uses a purely AST-based approach. Running unparam 




##TODO
- running on multiple files?	

## Contributing

Pull requests welcome!


## Other static analysis tools

If you've enjoyed nakedret, take a look at my other static anaylsis tools!

- [unimport](https://github.com/alexkohler/unimport) - Finds unnecessary import aliases
- [prealloc](https://github.com/alexkohler/prealloc) - Finds slice declarations that could potentially be preallocated.


