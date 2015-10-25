// Copyright (c) 2014, Rob Thornton
// All rights reserved.
// This source code is governed by a Simplied BSD-License. Please see the
// LICENSE included in this distribution for a copy of the full license
// or, if one is not included, you may also find a copy at
// http://opensource.org/licenses/BSD-2-Clause

// Package comp comprises the code generation portion of the Calc
// programming language
package comp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rthornton128/calc/ast"
	"github.com/rthornton128/calc/ir"
	"github.com/rthornton128/calc/parse"
	"github.com/rthornton128/calc/token"
)

type compiler struct {
	fp       *os.File
	fset     *token.FileSet
	errors   token.ErrorList
	nextID   int
	curScope *ir.Scope
}

// CompileFile generates a C source file for the corresponding file
// specified by path. The .calc extension for the filename in path is
// replaced with .c for the C source output.
func CompileFile(path string, opt bool) error {
	var c compiler

	c.fset = token.NewFileSet()
	f, err := parse.ParseFile(c.fset, path, nil)
	if err != nil {
		return err
	}

	pkg := ir.MakePackage(&ast.Package{
		Scope: ast.NewScope(nil),
		Files: []*ast.File{f},
	}, filepath.Base(path))
	ir.TypeCheck(pkg)
	if opt {
		pkg = ir.FoldConstants(pkg)
	}
	ir.Tag(pkg)

	path = path[:len(path)-len(filepath.Ext(path))]
	fp, err := os.Create(path + ".c")
	if err != nil {
		return err
	}
	defer fp.Close()

	c.fp = fp
	c.nextID = 1

	c.emitHeaders()
	c.compPackage(pkg)
	c.emitMain()

	if c.errors.Count() != 0 {
		return c.errors
	}
	return nil
}

// CompileDir generates C source code for the Calc sources found in the
// directory specified by path. The C source file uses the same name as
// directory rather than any individual file.
func CompileDir(path string, opt bool) error {
	fs := token.NewFileSet()
	p, err := parse.ParseDir(fs, path)
	if err != nil {
		return err
	}

	pkg := ir.MakePackage(p, filepath.Base(path))
	ir.TypeCheck(pkg)
	if opt {
		pkg = ir.FoldConstants(pkg)
	}
	ir.Tag(pkg)

	fp, err := os.Create(filepath.Join(path, filepath.Base(path)) + ".c")
	if err != nil {
		return err
	}
	defer fp.Close()

	c := &compiler{fp: fp, fset: fs, nextID: 1}

	c.emitHeaders()
	c.compPackage(pkg)
	c.emitMain()

	if c.errors.Count() != 0 {
		return c.errors
	}
	return nil
}

/* Utility */

func cType(t ir.Type) string {
	switch t {
	case ir.Int:
		return "int32_t"
	case ir.Bool:
		return "bool"
	default:
		return "int"
	}
}

// Error adds an error to the compiler at the given position. The remaining
// arguments are used to generate the error message.
func (c *compiler) Error(pos token.Pos, args ...interface{}) {
	c.errors.Add(c.fset.Position(pos), args...)
}

func (c *compiler) emit(s string, args ...interface{}) {
	fmt.Fprintf(c.fp, s, args...)
}

func (c *compiler) emitln(args ...interface{}) {
	fmt.Fprintln(c.fp, args...)
}

func (c *compiler) emitHeaders() {
	c.emitln("#include <stdio.h>")
	c.emitln("#include <stdint.h>")
	c.emitln("#include <stdbool.h>")
}

func (c *compiler) emitMain() {
	c.emitln("int main(void) {")
	c.emitln("printf(\"%d\\n\", _main());")
	c.emitln("return 0;")
	c.emitln("}")
}

/* Main Compiler */

func (c *compiler) compObject(o ir.Object) string {
	var str string
	switch t := o.(type) {
	case *ir.Assignment:
		c.compAssignment(t)
	case *ir.Constant:
		str = c.compConstant(t)
	case *ir.Binary:
		str = c.compBinary(t)
	case *ir.Call:
		str = c.compCall(t)
	case *ir.Declaration:
		c.compDeclaration(t)
	case *ir.Block:
		for _, e := range t.Exprs {
			str = c.compObject(e)
		}
	case *ir.If:
		str = c.compIf(t)
	case *ir.Unary:
		str = c.compUnary(t)
	case *ir.Var:
		str = c.compVar(t)
	case *ir.Variable:
		c.compVariable(t)
	}
	return str
}

func (c *compiler) compAssignment(a *ir.Assignment) {
	o := a.Scope().Lookup(a.Lhs)
	c.emit("%s = %s;\n", c.compObject(o), c.compObject(a.Rhs))
}

func (c *compiler) compBinary(b *ir.Binary) string {
	c.emit("%s _v%d = %s %s %s;\n", cType(b.Type()), b.ID(),
		c.compObject(b.Lhs), b.Op.String(), c.compObject(b.Rhs))
	return fmt.Sprintf("_v%d", b.ID())
}

func (c *compiler) compCall(call *ir.Call) string {
	args := make([]string, len(call.Args))
	for i, a := range call.Args {
		args[i] = fmt.Sprintf("%s", c.compObject(a))
	}
	return fmt.Sprintf("_%s(%s)", call.Name(), strings.Join(args, ","))
}

func (c *compiler) compConstant(con *ir.Constant) string {
	return con.String()
}

func (c *compiler) compDeclaration(d *ir.Declaration) {
	c.emit("%s {\n", c.compSignature(d))
	c.emit("return %s;\n}\n", c.compObject(d.Body))
}

func (c *compiler) compIdent(i *ir.Var) string {
	return fmt.Sprintf("_v%d", i.Scope().Lookup(i.Name()).(ir.IDer).ID())
}

func (c *compiler) compIf(i *ir.If) string {
	c.emit("%s _v%d = 0;\n", cType(i.Type()), i.ID())
	c.emit("if (%s) {\n", c.compObject(i.Cond))
	c.emit("_v%d = %s;\n", i.ID(), c.compObject(i.Then))
	if i.Else != nil {
		c.emitln("} else {")
		c.emit("_v%d = %s;\n", i.ID(), c.compObject(i.Else))
	}
	c.emitln("}")
	return fmt.Sprintf("_v%d", i.ID())
}

func (c *compiler) compPackage(p *ir.Package) {
	names := p.Scope().Names()
	for _, name := range names {
		d := p.Scope().Lookup(name).(*ir.Declaration)
		c.emit("%s;\n", c.compSignature(d))
		defer c.compDeclaration(d)
	}
}

func (c *compiler) compSignature(d *ir.Declaration) string {
	params := make([]string, len(d.Params))
	for i, p := range d.Params {
		param := d.Scope().Lookup(p).(*ir.Param)
		params[i] = fmt.Sprintf("%s _v%d", cType(param.Type()), param.ID())
	}
	return fmt.Sprintf("%s _%s(%s)", cType(d.Type()), d.Name(),
		strings.Join(params, ","))
}

func (c *compiler) compUnary(u *ir.Unary) string {
	return fmt.Sprintf("%s%s", u.Op, c.compObject(u.Rhs))
}

func (c *compiler) compVar(v *ir.Var) string {
	return fmt.Sprintf("_v%d", v.Scope().Lookup(v.Name()).(ir.IDer).ID())
}

func (c *compiler) compVariable(v *ir.Variable) {
	c.emit("%s _v%d = 0;\n", cType(v.Type()), v.ID())
	if v.Assign != nil {
		c.compObject(v.Assign)
	}
}
