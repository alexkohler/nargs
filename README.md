# nargs

nargs is a Go static analysis tool to find unused arguments in function declarations.

## Installation

    go get -u github.com/alexkohler/nargs/cmd/nargs

## Usage

Similar to other Go static anaylsis tools (such as golint, go vet), nargs can be invoked with one or more filenames, directories, or packages named by its import path. nargs also supports the `...` wildcard. 

    nargs [flags] files/directories/packages
	
### Flags
- **-tests** (default true) - Include test files in analysis
- **-set_exit_status** (default true) - Set exit status to 1 if any issues are found.

## Purpose

Often, parameters will be added to functions (such as a constructor), and then not actually used within the function. This tools was written to flag these types of functions to encourage either removing the parameters or using the blank identifier "_" to indicate that the parameter is not used.

## Examples

```Go
// test.go
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
```

```Bash
$ nargs main.go 
test.go:5 funcOne contains unused parameter c
test.go:12 funcTwo contains unused parameter c
test.go:17 funcThree contains unused parameter z
test.go:17 funcThree contains unused parameter recv
test.go:22 funcFour contains unused parameter namedReturn
```

## FAQ

### How is this different than [unparam](https://github.com/mvdan/unparam)?

`unparam` errs on the safe side to minimize false positives (ignoring functions that satisfy an interface, etc.). `nargs` takes a more aggressive approach and encourages the use of the blank identifier `_` for function parameters that are intentionally not used. `unparam` operates using the [ssa](https://godoc.org/golang.org/x/tools/go/ssa) package, whereas `nargs` uses a purely AST-based approach. Running unparam on the example file above only finds the issue in funcOne.

```Bash
$ unparam test.go 
test.go:5:28: c is unused
```


### How should these issues be fixed?

In simple cases, the arguments can simply be removed. nargs ignores function variables using the blank identifier `_`, and encourages the use of the blank identifier in the event that the parameter cannot be removed from the function due to implementing an interface or function typedef. If this is the case, the following can be done to fix the above example:

```Go
package main

func funcOne(a int, b int, _ int) int {
        return a + b
}

type f struct{}

func (f) funcTwo(a int, b int, _ int) int {
        return a + b
}

func (f) funcThree(_ int) int {
        return 5
}

func funcFour() (namedReturn int) {
```

## Other static analysis tools

If you've enjoyed nargs, take a look at my other static anaylsis tools!

- [prealloc](https://github.com/alexkohler/prealloc) - Finds slice declarations that could potentially be preallocated.
- [nakedret](https://github.com/alexkohler/nakedret) - Finds naked returns.
- [identypo](https://github.com/alexkohler/identypo) - Finds typos in identifiers (functions, function calls, variables, constants, type declarations, packages, labels) including CamelCased functions, variables, etc. 
- [unimport](https://github.com/alexkohler/unimport) - Finds unnecessary import aliases
- [dogsled](https://github.com/alexkohler/dogsled) - Finds assignments/declarations with too many blank identifiers (e.g. x, _, _, _, := f()).


