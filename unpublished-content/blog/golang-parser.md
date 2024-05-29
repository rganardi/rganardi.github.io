---
title: "Golang Parser"
description: "a tour into go parser and other go internals"
date: 2021-12-29T23:07:06+01:00
draft: true
---

Inspired by [this wonderful post](http://mathamy.com/import-accio-bootstrapping-python-grammar.html), I wanted to make something similar. I've been bugged by the fact that go just *has to* use `nil` instead of the standard `null`, so let's fix that.

Now, a perfectly reasonable approach to this would be to
```
$ cd ~/go/src/
$ find -name '*.go' | xargs -I{} sed -ie 's/nil/null/g' {}
```
and be done with it.

However, this throws the following error
```
Building Go cmd/dist using /usr/lib/go. (go1.17.5 linux/amd64)
# github.com/golang/go/src/cmd/dist
cmd/dist/build.go:249:100: undefined: null
cmd/dist/build.go:422:11: undefined: null
cmd/dist/build.go:600:13: undefined: null
cmd/dist/build.go:601:16: undefined: null
cmd/dist/build.go:602:16: undefined: null
cmd/dist/build.go:603:16: undefined: null
cmd/dist/build.go:604:16: undefined: null
cmd/dist/build.go:619:11: undefined: null
cmd/dist/build.go:785:17: undefined: null
cmd/dist/build.go:874:38: undefined: null
cmd/dist/build.go:874:38: too many errors
```
This is because there are many `nil` in the go source code.
Also, it doesn't seem fair, and we don't get to learn anything from it.

So, let's take the hard route.

# compiler theory

Compilers usually works in several phases:
 1. Lexer
 2. Parser
 3. Code generator

Each of these phases has different responsibilities.

 1.
	Lexer's job is to split the input into a series of tokens.
	In real terms, that's
	```go {linenos=false}
	func lex([]byte) ([]token)
	```

 2.
	Parser's job is to create a parse tree (or sometimes called abstract syntax tree) from the series of tokens.
	This is where you have your LALR(k) parsers, LR(k) parsers, etc.
	The function signature is
	```go {linenos=false}
	func parse([]token) (*ast)
	```

 3.
	Code generator's job is to make sense out of the `ast`.
	What that means depends on if the language is interpreted or a compiled language.
	For a compiler targetting machine code like go, here you translate the ast to a series of machine code.
	For a compiler targetting another language (like typescript compiler), here you output the destination language.

	The function signature is
	```go {linenos=false}
	func generate(*ast) ([]byte)
	```

Sometimes, there are actually two steps of code generation.
First the compiler will generate an intermediate representation (IR) of the AST.
Then we generate the target from the IR.

The reason for this is twofold:

 * Manipulating the IR is often easier than manipulating the AST. So some compilers will optimize at this stage instead of before.

 * If the compiler have many targets, then it makes more sense to write one compiler giving an IR, then write different backends that generate machine code from that IR.

Now, our goal is to change `nil` to `null`.
In theory, changing the lexer is enough to do that, because if we just change the token from `nil` to `null`, everything else should work the same way.

# lexer

In principle, this is the only part that we'll need to touch.
If the go compiler follows the usual architecture, we can simply change the keyword for the `nil` token.
Let's see some stuff in go lexer code.

Go's usual convention is to have a package deal with all aspects of your application, then the actual executable code is in the `cmd/` directory.
If we check out the go compiler [source code](https://github.com/golang/go/archive/master.zip), we actually the same structure there.

```
$ curl -LO https://github.com/golang/go/archive/master.zip
$ unzip master.zip
$ cd go-master
$ cd src
$ ls
***snip***
bytes/
cmd/
compress/
***snip***
```

First off, let us build and check the version

```
$ ./make.bash
Building Go cmd/dist using /usr/lib/go. (go1.17.5 linux/amd64)
Building Go toolchain1 using /usr/lib/go.
Building Go bootstrap cmd/go (go_bootstrap) using Go toolchain1.
Building Go toolchain2 using go_bootstrap and Go toolchain1.
Building Go toolchain3 using go_bootstrap and Go toolchain2.
Building packages and commands for linux/amd64.
---
Installed Go for linux/amd64 in $GOPATH/src/github.com/golang/go
Installed commands in $GOPATH/src/github.com/golang/go/bin
$ $(go env GOPATH)/src/github.com/golang/go/bin/go version
go version devel go1.18-b357b05 Thu Dec 23 20:03:38 2021 +0000 linux/amd64
```

I'm working with version 1.17.5 on my machine, and the last commit on github bundle that I downloaded is `b357b05b70d2b8c4988ac2a27f2af176e7a09e1b`.

The `go` binary entry point is located in [`src/cmd/go/main.go`](https://github.com/golang/go/blob/b357b05b70d2b8c4988ac2a27f2af176e7a09e1b/src/cmd/go/main.go#L46), so let's check that out.

Right off the bat you see

```go {linenos=false}
// src/cmd/go/main.go

func init() {
	base.Go.Commands = []*base.Command{
		bug.CmdBug,
		work.CmdBuild,
		clean.CmdClean,
		doc.CmdDoc,
		envcmd.CmdEnv,
```

This looks like a list of all go subcommands, as we can verify when we run `go`.

```
$ go
Go is a tool for managing Go source code.

Usage:

        go <command> [arguments]

The commands are:

        bug         start a bug report
        build       compile packages and dependencies
        clean       remove object files and cached files
        doc         show documentation for package or symbol
        env         print Go environment information
```

We're interested in the build command, so let's follow that.

```go {linenos=false}
// src/cmd/go/internal/work/build.go

var CmdBuild = &base.Command{
	UsageLine: "go build [-o output] [-i] [build flags] [packages]",
	Short:     "compile packages and dependencies",
```

However, scanning through [`src/cmd/go/internal/work/build.go`](https://github.com/golang/go/blob/b357b05b70d2b8c4988ac2a27f2af176e7a09e1b/src/cmd/go/internal/work/build.go#L392), it seems like it's a lot of framework code.
It looks like the actual build happens here

```go {linenos=false}
// src/cmd/go/internal/work/build.go

func runBuild(ctx context.Context, cmd *base.Command, args []string) {
	modload.InitWorkfile()
	BuildInit()
	var b Builder

	***snip***

	b.Do(ctx, a)
}
```

which leads to

```go {linenos=false}
// src/cmd/go/internal/work/exec.go
func (b *Builder) Do(ctx context.Context, root *Action) {
```

This function looks like a task-management system.
If we trace the function, it looks like the work is performed by this call

```go {linenos=false}
// src/cmd/go/internal/work/exec.go
func (b *Builder) Do(ctx context.Context, root *Action) {
			***snip***
			err = a.Func(b, ctx, a)
			***snip***
```

with `a` being some action object.
So we need to trace what is the function attached to this action.
Going back to [`src/cmd/go/internal/work/build.go`](https://github.com/golang/go/blob/b357b05b70d2b8c4988ac2a27f2af176e7a09e1b/src/cmd/go/internal/work/build.go#L479), we see

```go {linenos=false}
	a := &Action{Mode: "go build"}
	for _, p := range pkgs {
		a.Deps = append(a.Deps, b.AutoAction(ModeBuild, depMode, p))
	}
```

which leads to

```go {linenos=false}
// CompileAction returns the action for compiling and possibly installing
// (according to mode) the given package. The resulting action is only
// for building packages (archives), never for linking executables.
// depMode is the action (build or install) to use when building dependencies.
// To turn package main into an executable, call b.Link instead.
func (b *Builder) CompileAction(mode, depMode BuildMode, p *load.Package) *Action {
		***snip***
		a := &Action{
			Mode:    "build",
			Package: p,
			Func:    (*Builder).build,
			Objdir:  b.NewObjdir(),
		}
		***snip***
```

BINGO!
So it calls the `(*Builder).build` method to perform the actual work.
This is defined in [`src/cmd/go/internal/work/exec.go`](https://github.com/golang/go/blob/b357b05b70d2b8c4988ac2a27f2af176e7a09e1b/src/cmd/go/internal/work/exec.go#L455)

```go {linenos=false}
// src/cmd/go/internal/work/exec.go

func (b *Builder) build(ctx context.Context, a *Action) (err error) {
```

This looks like the guy who's doing the actual heavy-lifting, calling gccgo and SWIG if needed.
However, this seems to be going too deep, since I can't see any call to the lexer or the parser.
Let us check what kind of error does using `null` gives

```
$ cat << EOF > test.go
// test.go

package main

import "fmt"

func main() {
	fmt.Println("hello!", null)
	return
}
EOF
$ go run test.go
# command-line-arguments
./test.go:8:24: undefined: null
```
Huh!
It does not give a syntax error!
I guess the lexer and the parser must have worked then, and it's only in the later stages that the compiler throws an error.

# parser

Since lexer was the wrong place to look at, let's look at the next step.
First we set up a testing script to see the parse tree.

```go
// parse.go

package main

import (
	"fmt"
	"os"

	"abc/syntax"
)

func debug(v interface{}) {
	fmt.Printf("%T: %+v\n", v, v)
}

func process(q syntax.Node) {
	switch t := q.(type) {
	case *syntax.File:
		for _, p := range t.DeclList {
			process(p)
		}
	default:
		debug(t)
	}
}

func main() {
	fname := "test.go"
	fbase := syntax.NewFileBase(fname)

	f, err := os.Open(fname)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	mode := syntax.CheckBranches

	p, err := syntax.Parse(fbase, f, nil, nil, mode)
	if err != nil {
		fmt.Println(err)
		return
	}

	process(p)

	return
}
```

The main parse step is done through the `syntax.Parse` call in line 40.

To fool the module system, we'll also set the script as a module and soft link the `github.com/cmd/compile/internal/syntax` package into the module

```
$ cat > go.mod << EOF
module abc

go 1.17
EOF
$ ln -s $(go env GOPATH)/src/github.com/golang/go/src/cmd/compile/internal/syntax
```

Running `go run parse.go`, we get

```
$ go run parse.go
*syntax.ImportDecl: &{Group:<nil> Pragma:<nil> LocalPkgName:<nil> Path:0xc000066270 decl:{node:{pos:{base:0xc000066240 line:5 col:8}}}}
*syntax.FuncDecl: &{Pragma:<nil> Recv:<nil> Name:0xc0000200c0 TParamList:[] Type:0xc000072100 Body:0xc000072140 decl:{node:{pos:{base:0xc000066240 line:7 col:6}}}}
```

We see that there are two nodes, `ImportDecl` and `FuncDecl`.
Clearly, `ImportDecl` refers to the `import "fmt"` statement, and `FuncDecl` is the definition of `main()`.
So let's print the body of the function, since `nil` is inside the body.
We simply add the following clause to the switch in `process()`

```go {linenos=false}
	case *syntax.FuncDecl:
		fmt.Printf("\nfunction ")
		debug(t.Name.Value)
		for _, s := range t.Body.List {
			process(s)
		}
```

which gives us

```
$ go run parse.go
*syntax.ImportDecl: &{Group:<nil> Pragma:<nil> LocalPkgName:<nil> Path:0xc0000a0270 decl:{node:{pos:{base:0xc0000a0240 line:5 col:8}}}}

function string: main
*syntax.ExprStmt: &{X:0xc0000b6180 simpleStmt:{stmt:{node:{pos:{base:0xc0000a0240 line:9 col:13}}}}}
*syntax.ReturnStmt: &{Results:<nil> stmt:{node:{pos:{base:0xc0000a0240 line:10 col:2}}}}
```

As expected, the `FuncDecl` was the declaration of `main()`.
Now it's just a matter of going down the parse tree

```go {linenos=false}
	case *syntax.ExprStmt:
		process(t.X)
	case *syntax.CallExpr:
		debug("call")
		process(t.Fun)
		for _, a := range t.ArgList {
			process(a)
		}
```

until

```
$ go run parse.go
*syntax.ImportDecl: &{Group:<nil> Pragma:<nil> LocalPkgName:<nil> Path:0xc0000a0270 decl:{node:{pos:{base:0xc0000a0240 line:5 col:8}}}}

function string: main
string: call
*syntax.SelectorExpr: &{X:0xc0000be0a0 Sel:0xc0000be0c0 expr:{node:{pos:{base:0xc0000a0240 line:9 col:5}}}}
*syntax.BasicLit: &{Value:"hello!" Kind:4 Bad:false expr:{node:{pos:{base:0xc0000a0240 line:9 col:14}}}}
*syntax.Name: &{Value:null expr:{node:{pos:{base:0xc0000a0240 line:9 col:24}}}}
*syntax.ReturnStmt: &{Results:<nil> stmt:{node:{pos:{base:0xc0000a0240 line:10 col:2}}}}
```

*HEY!* There's a `syntax.Name` node with `Value: null`!
Looks like the value `nil` is represented as a `syntax.Name` node in the parse tree, with the field `Value` giving the name.
Therefore, all we need to do is to recognize the value `null` and generate the same node as `nil` when encountered.

Conveniently, there is already a function that produces a `syntax.Name` node.

```go {linenos=false}
// src/cmd/compile/internal/syntax/nodes.go

func NewName(pos Pos, value string) *Name {
	n := new(Name)
	n.pos = pos
	n.Value = value
	return n
}
```

All we need to do is to patch it to recognize `null`,

```go {linenos=false}
// src/cmd/compile/internal/syntax/nodes.go

func NewName(pos Pos, value string) *Name {
	n := new(Name)
	n.pos = pos
	n.Value = value
	if value == "null" {
		n.Value = "nil"
	}
	return n
}
```

build (`-a` to force rebuild, `-v` to verify the syntax package *is* actually rebuilt), and 

```
$ go run -a -v parse.go
***snip***
abc/syntax
command-line-arguments
*syntax.ImportDecl: &{Group:<nil> Pragma:<nil> LocalPkgName:<nil> Path:0xc0000a0270 decl:{node:{pos:{base:0xc0000a0240 line:5 col:8}}}}

function string: main
string: call
*syntax.SelectorExpr: &{X:0xc0000be0a0 Sel:0xc0000be0c0 expr:{node:{pos:{base:0xc0000a0240 line:8 col:5}}}}
*syntax.BasicLit: &{Value:"hello!" Kind:4 Bad:false expr:{node:{pos:{base:0xc0000a0240 line:8 col:14}}}}
*syntax.Name: &{Value:nil expr:{node:{pos:{base:0xc0000a0240 line:8 col:24}}}}
*syntax.ReturnStmt: &{Results:<nil> stmt:{node:{pos:{base:0xc0000a0240 line:9 col:2}}}}
```

we have succeeded!

Now we only have to build the go binary and we can use `null` instead of `nil`!

```
$ ./make.bash
$ $(go env GOPATH)/src/github.com/golang/go/bin/go run test.go
hello! <nil>
```

# last words

We have fixed go by using `null` instead of `nil`.
Well, we didn't really, because our compiler still recognizes `nil`
We can make it *not* recognize `nil`, but it will be a headache because of bootstrapping.
That means some parts of the source for the go compiler is written in go, and we need to change all occurrences of `nil` to `null` in the whole codebase, which is just a lot of work.
But otherwise, we have succeeded!

Congrats!
