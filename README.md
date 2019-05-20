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
- **-receivers** (default true) - Report unused function receivers.


## Purpose

Often, parameters will be added to functions (such as a constructor), and then not actually used within the function. This tools was written to flag these types of functions to encourage either removing the parameters or using the blank identifier "_" to indicate that the parameter is intentionally not used.

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
```

```Bash
$ nargs -named-returns=true main.go 
test.go:5 funcOne contains unused parameter c
test.go:12 funcTwo contains unused parameter c
test.go:17 funcThree contains unused parameter recv
test.go:22 funcFour contains unused parameter namedReturn
```

## FAQ

### How is this different than [unparam](https://github.com/mvdan/unparam)?

By design, `unparam` errs on the safe side to minimize false positives (ignoring functions that potentially satisfy an interface, etc.). `nargs` takes a more aggressive approach and encourages the use of the blank identifier `_` for function parameters that are intentionally not used. `unparam` operates using the [ssa](https://godoc.org/golang.org/x/tools/go/ssa) package, whereas `nargs` uses a purely AST-based approach. Running unparam on the example file above only finds the issue in funcOne. funcTwo and funcThree are ignored due to potentially implmenting an interface.

```Bash
$ unparam test.go 
test.go:5:28: c is unused
```


### How should these issues be fixed?

If the function is implementing an interface or function typedef, the blank identifier `_` should be used and `nargs` will no longer flag the parameter as being unused. In other cases, the arguments can simply be removed. Suppose all the functions from our example above were implementing an interface or function typedef. Then, the following can be done to fix the above example:

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
        return
}
```

## Other static analysis tools

If you've enjoyed nargs, take a look at my other static anaylsis tools!

- [prealloc](https://github.com/alexkohler/prealloc) - Finds slice declarations that could potentially be preallocated.
- [nakedret](https://github.com/alexkohler/nakedret) - Finds naked returns.
- [identypo](https://github.com/alexkohler/identypo) - Finds typos in identifiers (functions, function calls, variables, constants, type declarations, packages, labels) including CamelCased functions, variables, etc. 
- [unimport](https://github.com/alexkohler/unimport) - Finds unnecessary import aliases.
- [dogsled](https://github.com/alexkohler/dogsled) - Finds assignments/declarations with too many blank identifiers (e.g. x, _, _, _, := f()).


