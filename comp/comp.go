package comp

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rthornton128/calc1/ast"
	"github.com/rthornton128/calc1/parse"
	"github.com/rthornton128/calc1/token"
)

type compiler struct {
	fp *os.File
}

func CompileFile(fname, src string) {

	var c compiler
	fp, err := os.Create(fname + ".c")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	f := parse.ParseFile(fname, src)
	c.fp = fp
	c.compFile(f)
}

func (c *compiler) compNode(node ast.Node) int {
	switch n := node.(type) {
	case *ast.BasicLit:
		i, err := strconv.Atoi(n.Lit)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return i
	case *ast.BinaryExpr:
		return c.compBinaryExpr(n)
	default:
		return 0 /* can't be reached */
	}
}

func (c *compiler) compBinaryExpr(b *ast.BinaryExpr) int {
	var tmp int

	tmp = c.compNode(b.List[0])

	for _, node := range b.List[1:] {
		switch b.Op {
		case token.ADD:
			tmp += c.compNode(node)
		case token.SUB:
			tmp -= c.compNode(node)
		case token.MUL:
			tmp *= c.compNode(node)
		case token.QUO:
			tmp /= c.compNode(node)
		case token.REM:
			tmp %= c.compNode(node)
		}
	}

	return tmp
}

func (c *compiler) compFile(f *ast.File) {
	fmt.Fprintln(c.fp, "#include <stdio.h>")
	fmt.Fprintln(c.fp, "int main(void) {")
	fmt.Fprintf(c.fp, "printf(\"%%d\", %d);\n", c.compNode(f.Root))
	fmt.Fprintln(c.fp, "return 0;")
	fmt.Fprintln(c.fp, "}")
}
