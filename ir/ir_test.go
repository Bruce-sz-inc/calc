package ir_test

import (
	"testing"

	"github.com/rthornton128/calc/ast"
	"github.com/rthornton128/calc/ir"
	"github.com/rthornton128/calc/parse"
)

func TestPrint(t *testing.T) {
	test_handler(t, "example1", "(decl main (a b int) int 42)")
	test_handler(t, "example2", "(decl main int (fn 2 3))")
	test_handler(t, "example3", "(decl main int (+ 2 3))")
	test_handler(t, "example4", "(decl main int (+ 2 3 4 5))")
	test_handler(t, "example5", "(decl main int -24)")
	test_handler(t, "example6", "(decl main int ((= a 42) a))")
	test_handler(t, "example7", "(decl main int (if (== 1 1) int 1 0))")
	test_handler(t, "example8", "(decl main int ((var a int) a))")
	test_handler(t, "example9", "(decl main int ((var (= a 42)) a))")
}

func test_handler(t *testing.T, name, src string) {
	n, err := parse.ParseExpression(name, src)
	if err != nil {
		t.Fatal(err)
	}
	s := ast.NewScope(nil)
	pkg := &ast.Package{
		Scope: s,
		Files: []*ast.File{&ast.File{
			Decls: []*ast.DeclExpr{n.(*ast.DeclExpr)},
			Scope: ast.NewScope(s)},
		},
	}

	p := ir.MakePackage(pkg, name)
	t.Log(p)
	t.Log(ir.FoldConstants(p))
	ir.Tag(p)

}
