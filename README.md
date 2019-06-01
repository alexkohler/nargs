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
- **-named_returns** (default false) - Report unused named return arguments. This is false by default because named returns can be used to provide context to what's being returned.
- **-receivers** (default false) - Report unused function receivers. This is false by default because it would otherwise generate a fair number of false positives, depending on your coding standard.


## Purpose

Often, parameters will be added to functions (such as a constructor), and then not actually used within the function. This tool was written to flag these types of functions to encourage either removing the parameters or using the blank identifier `_` to indicate that the parameter is intentionally not used. It's worth noting that this linter is aggressive by design and may have false positives.

## Examples

```Go
// test.go
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
func closure() {
	c := func(v int) {
		enclosed := 2
		enclosed++
	}
	c(1)
}

// Unused function as parameter
func unusedFunc(f func()) {
}

// Unused closure parameter in package scoped closure
var z = func(i int) {
	fmt.Println()
}
```

```Bash
$ $ nargs testdata/test.go 
testdata/test.go:6 funcOne contains unused parameter c
testdata/test.go:13 funcTwo contains unused parameter c
testdata/test.go:31 c contains unused parameter v
testdata/test.go:39 unusedFunc contains unused parameter f
testdata/test.go:43 z contains unused parameter i
```

## FAQ

### How is this different than [unparam](https://github.com/mvdan/unparam)?

By design, `unparam` errs on the safe side to minimize false positives (ignoring functions that potentially satisfy an interface or function typedef, etc.). `nargs` takes a more aggressive approach and encourages the use of the blank identifier `_` for function parameters that are intentionally not used. `unparam` operates using the [ssa](https://godoc.org/golang.org/x/tools/go/ssa) package, whereas `nargs` uses a purely AST-based approach. Running unparam on the example file above only finds the issue in funcOne. funcTwo and funcThree are ignored due to potentially implementing an interface. 

```Bash
$ unparam test.go 
test.go:5:28: c is unused
```


### How should these issues be fixed?

If the function is implementing an interface or function typedef, the blank identifier `_` should be used and `nargs` will no longer flag the parameter as being unused. In other cases, the arguments can simply be removed. Suppose funcOne from our example above could not be removed due to meeting a function typedef. In this case, the following can be done to fix the above example:

```Go
package main

// testdata/test.go:6 funcOne contains unused parameter c - use '_' on the 'c' parameter
func funcOne(a int, b int, _ int) int {
        return a + b
}
```


## Other static analysis tools

If you've enjoyed nargs, take a look at my other static anaylsis tools!

- [prealloc](https://github.com/alexkohler/prealloc) - Finds slice declarations that could potentially be preallocated.
- [nakedret](https://github.com/alexkohler/nakedret) - Finds naked returns.
- [identypo](https://github.com/alexkohler/identypo) - Finds typos in identifiers (functions, function calls, variables, constants, type declarations, packages, labels) including CamelCased functions, variables, etc. 
- [unimport](https://github.com/alexkohler/unimport) - Finds unnecessary import aliases.
- [dogsled](https://github.com/alexkohler/dogsled) - Finds assignments/declarations with too many blank identifiers (e.g. x, _, _, _, := f()).


