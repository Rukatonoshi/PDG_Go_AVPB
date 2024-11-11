package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strings"

	"golang.org/x/tools/go/cfg"
)

func printCFG(cg *cfg.CFG) {
	for _, block := range cg.Blocks {
		fmt.Printf("Block: %s\n", block.String())
		for _, node := range block.Nodes {
			switch n := node.(type) {
			case *ast.DeclStmt:
				printValueSpec(n.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec))
			case *ast.AssignStmt:
				printAssignStmt(n)
			case *ast.ReturnStmt:
				printReturnStmt(n)
			case *ast.ExprStmt:
				printExpr(n.X)
			case *ast.ParenExpr:
				printExpr(n.X)
			case *ast.IncDecStmt:
				printIncDecStmt(n)
			case *ast.BinaryExpr:
				printBinaryExpr(n)
			case *ast.CallExpr:
				printCallExpr(n)
			default:
				fmt.Printf(" -> Node (Unhandled): %T\n", node)
			}
		}
		for _, succ := range block.Succs {
			fmt.Printf(" -> Successor: %s\n", succ.String())
		}
	}
}

func printValueSpec(valueSpec *ast.ValueSpec) {
	for i, name := range valueSpec.Names {
		value := "nil"
		if i < len(valueSpec.Values) {
			value = getValue(valueSpec.Values[i])
		}
		fmt.Printf(" -> Node: %s = %s\n", name.Name, value)
	}
}

func printAssignStmt(assignStmt *ast.AssignStmt) {
	for i, lhs := range assignStmt.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			value := "nil"
			if i < len(assignStmt.Rhs) {
				value = getValue(assignStmt.Rhs[i])
			}
			fmt.Printf(" -> Node: %s = %s\n", ident.Name, value)
		}
	}
}

func printReturnStmt(returnStmt *ast.ReturnStmt) {
	values := []string{}
	for _, result := range returnStmt.Results {
		values = append(values, getValue(result))
	}
	fmt.Printf(" -> Return: %s\n", values)
}

func printExpr(expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		printBinaryExpr(e)
	case *ast.CallExpr:
		printCallExpr(e)
	case *ast.ParenExpr:
		fmt.Printf(" -> Node: (%s)\n", getValue(e.X))
	default:
		fmt.Printf(" -> Node (Unhandled Expr): %T\n", expr)
	}
}

func printIncDecStmt(incDecStmt *ast.IncDecStmt) {
	fmt.Printf(" -> Node: %s %s\n", incDecStmt.X.(*ast.Ident).Name, incDecStmt.Tok.String())
}

func printBinaryExpr(binaryExpr *ast.BinaryExpr) {
	fmt.Printf(" -> Node: %s %s %s\n", getValue(binaryExpr.X), binaryExpr.Op.String(), getValue(binaryExpr.Y))
}

func printCallExpr(callExpr *ast.CallExpr) {
	funcName := getValue(callExpr.Fun)
	args := []string{}
	for _, arg := range callExpr.Args {
		args = append(args, getValue(arg))
	}
	fmt.Printf(" -> Node: %s(%s)\n", funcName, args)
}

func getValue(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", getValue(e.X), e.Op.String(), getValue(e.Y))
	case *ast.CallExpr:
		if ident, ok := e.Fun.(*ast.Ident); ok {
			return ident.Name
		}
		return "function call"
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", getValue(e.X), e.Sel.Name)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func genDot(cg *cfg.CFG) string {
	dot := "digraph G {\n"
	variables := make(map[string][]string)
	for _, block := range cg.Blocks {
		if !block.Live {
			continue
		}
		blockID := fmt.Sprintf("block_%d", block.Index)
		dot += fmt.Sprintf("  %s [label=\"%s\"];\n", blockID, block.String())
		var prevNodeID string
		var lastNodeID string
		var loopID string
		if len(block.Nodes) == 0 {
			for _, succ := range block.Succs {
				succID := fmt.Sprintf("block_%d", succ.Index)
				color := "black"
				if succ.Kind == cfg.KindIfThen || succ.Kind == cfg.KindForBody {
					color = "purple"
				} else if succ.Kind == cfg.KindIfDone || succ.Kind == cfg.KindIfElse || succ.Kind == cfg.KindForDone {
					color = "red"
				}
				dot += fmt.Sprintf("  %s -> %s [color=\"%s\"];\n", blockID, succID, color)
			}
		}
		for i, node := range block.Nodes {
			nodeID := fmt.Sprintf("%s_node_%d", blockID, i)
			switch n := node.(type) {
			case *ast.DeclStmt:
				valueSpec := n.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
				for j, name := range valueSpec.Names {
					value := "nil"
					if j < len(valueSpec.Values) {
						value = getValue(valueSpec.Values[j])
					}
					dot += fmt.Sprintf("  %s [label=\"%s = %s\"];\n", nodeID, name.Name, value)
					variables[name.Name] = append(variables[name.Name], nodeID)
				}
			case *ast.AssignStmt:
				for j, lhs := range n.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						value := "nil"
						if j < len(n.Rhs) {
							value = getValue(n.Rhs[j])
						}
						dot += fmt.Sprintf("  %s [label=\"%s = %s\"];\n", nodeID, ident.Name, value)
						variables[ident.Name] = append(variables[ident.Name], nodeID)
					}
				}
			case *ast.ReturnStmt:
				values := []string{}
				for _, result := range n.Results {
					value := getValue(result)
					values = append(values, value)
					variables[value] = append(variables[value], nodeID)
				}
				dot += fmt.Sprintf("  %s [label=\"Return: %s\"];\n", nodeID, strings.Join(values, ", "))
			case *ast.ExprStmt:
				switch e := n.X.(type) {
				case *ast.BinaryExpr:
					dot += fmt.Sprintf("  %s [label=\"%s %s %s\"];\n", nodeID, getValue(e.X), e.Op.String(), getValue(e.Y))
				case *ast.CallExpr:
					funcName := getValue(e.Fun)
					args := []string{}
					for _, arg := range e.Args {
						args = append(args, getValue(arg))
					}
					label := fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", "))
					label = strings.ReplaceAll(label, `"`, `\"`) // escape double quotes
					dot += fmt.Sprintf("  %s [label=\"%s\"];\n", nodeID, label)
				default:
					dot += fmt.Sprintf("  %s [label=\"(Unhandled Expr): %T\"];\n", nodeID, n.X)
				}
			case *ast.IncDecStmt:
				varName := n.X.(*ast.Ident).Name
				dot += fmt.Sprintf("  %s [label=\"%s %s\"];\n", nodeID, n.X.(*ast.Ident).Name, n.Tok.String())
				variables[varName] = append(variables[varName], nodeID)
			case *ast.BinaryExpr:
				dot += fmt.Sprintf("  %s [label=\"%s %s %s\"];\n", nodeID, getValue(n.X), n.Op.String(), getValue(n.Y))
			case *ast.CallExpr:
				funcName := getValue(n.Fun)
				args := []string{}
				for _, arg := range n.Args {
					args = append(args, getValue(arg))
				}
				dot += fmt.Sprintf("  %s [label=\"%s(%s)\"];\n", nodeID, funcName, strings.Join(args, ", "))
			case *ast.SelectorExpr:
				dot += fmt.Sprintf("  %s [label=\"%s.%s\"];\n", nodeID, getValue(n.X), n.Sel.Name)
			case *ast.ParenExpr:
				if binaryExpr, ok := n.X.(*ast.BinaryExpr); ok {
					dot += fmt.Sprintf("  %s [label=\"%s %s %s\"];\n", nodeID, getValue(binaryExpr.X), binaryExpr.Op.String(), getValue(binaryExpr.Y))
				} else {
					dot += fmt.Sprintf("  %s [label=\"%s\"];\n", nodeID, getValue(n.X))
				}
			case *ast.IfStmt:
				cond := getValue(n.Cond)
				dot += fmt.Sprintf("  %s [label=\"if %s\"];\n", nodeID, cond)
				thenBlockID := fmt.Sprintf("block_%d", block.Succs[0].Index)
				dot += fmt.Sprintf("  %s -> %s [label=\"true\"];\n", nodeID, thenBlockID)
				if n.Else != nil {
					elseBlockID := fmt.Sprintf("block_%d", block.Succs[1].Index)
					dot += fmt.Sprintf("  %s -> %s [label=\"false\"];\n", nodeID, elseBlockID)
				}
			case *ast.ForStmt:
				loopID = nodeID
				cond := getValue(n.Cond)
				dot += fmt.Sprintf("  %s [label=\"for %s\"];\n", nodeID, cond)
				bodyBlockID := fmt.Sprintf("block_%d", block.Succs[0].Index)
				dot += fmt.Sprintf("  %s -> %s [label=\"true\"];\n", nodeID, bodyBlockID)
				postBlockID := fmt.Sprintf("block_%d", block.Succs[1].Index)
				dot += fmt.Sprintf("  %s -> %s [label=\"false\"];\n", nodeID, postBlockID)
			case *ast.BranchStmt:
				//				fmt.Printf("TESTING %s", n.Label.Obj)
				switch n.Tok {
				case token.CONTINUE:
					dot += fmt.Sprintf("  %s [label=\"continue\"];\n", nodeID)
					dot += fmt.Sprintf("  %s -> %s;\n", nodeID, loopID)
				case token.BREAK:
					dot += fmt.Sprintf("  %s [label=\"break\"];\n", nodeID)
				}
			default:
				fmt.Printf("Node type: %T ==> %s\n", node, nodeID) // debugging statement
				dot += fmt.Sprintf("  %s [label=\"(Unhandled): %T\"];\n", nodeID, node)
			}
			if prevNodeID != "" {
				dot += fmt.Sprintf("  %s -> %s;\n", prevNodeID, nodeID)
			} else {
				dot += fmt.Sprintf("  %s -> %s;\n", blockID, nodeID)
			}
			prevNodeID = nodeID
			lastNodeID = nodeID
		}
		for _, succ := range block.Succs {
			succID := fmt.Sprintf("block_%d", succ.Index)
			fmt.Printf("Block type: %s %d\n", succ.Kind, succ.Index) // debugging statement
			color := "black"
			if succ.Kind == cfg.KindIfThen || succ.Kind == cfg.KindForBody {
				color = "purple"
			} else if succ.Kind == cfg.KindIfDone || succ.Kind == cfg.KindIfElse || succ.Kind == cfg.KindForDone {
				color = "red"
			}

			if lastNodeID == "" {
				continue
			}
			dot += fmt.Sprintf("  %s -> %s [color=\"%s\"];\n", lastNodeID, succID, color)
		}
	}
	for varName, nodes := range variables {
		if len(nodes) > 1 {
			for i := 1; i < len(nodes); i++ {
				dot += fmt.Sprintf("  %s -> %s [label=\"%s\" style=dotted fontsize=26];\n", nodes[i-1], nodes[i], varName)
			}
		}
	}
	dot += "}\n"
	return dot
}

func main() {
	src := `
package main

func main() {
	testString = "Some string"
	fib(n)
}

func fib(int n) (c int) {
	a = 0
	b = 1

	if n < 2 {
		return n
	}

	for i := 1; i < n; i++ {
		c = a + b
		a = b
		if c == b {
			continue
		}
		b = c
	}
	return c
}
`
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, "example.go", src, parser.Trace)
	if err != nil {
		log.Fatalf("Error parsing source code: %v", err)
	}

	ast.Print(fset, node)

	fmt.Print("\n-------------------\n")
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Body != nil {
				predicate := func(*ast.CallExpr) bool { return true }
				cg := cfg.New(fn.Body, predicate)
				fmt.Printf("CFG for function: %s\n", fn.Name.Name)

				printCFG(cg)

				dotFmt := genDot(cg)

				fmt.Println("DOT Format:")
				fmt.Println(dotFmt)
			}
		}
	}
}
