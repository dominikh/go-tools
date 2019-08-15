package pattern

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
)

func ASTToNode(node interface{}) Node {
	switch node := node.(type) {
	case *ast.File:
		panic("cannot convert *ast.File to Node")
	case nil:
		return Nil{}
	case string:
		return String(node)
	case token.Token:
		return Token(node)
	case *ast.ExprStmt:
		return ASTToNode(node.X)
	case *ast.BlockStmt:
		if node == nil {
			return Nil{}
		}
		return ASTToNode(node.List)
	case *ast.FieldList:
		if node == nil {
			return Nil{}
		}
		return ASTToNode(node.List)
	case *ast.BasicLit:
		if node == nil {
			return Nil{}
		}
	case *ast.ParenExpr:
		return ASTToNode(node.X)
	}

	if node, ok := node.(ast.Node); ok {
		name := reflect.TypeOf(node).Elem().Name()
		T, ok := structNodes[name]
		if !ok {
			panic(fmt.Sprintf("internal error: unhandled type %T", node))
		}

		if reflect.ValueOf(node).IsNil() {
			return Nil{}
		}
		v := reflect.ValueOf(node).Elem()
		objs := make([]Node, T.NumField())
		for i := 0; i < T.NumField(); i++ {
			f := v.FieldByName(T.Field(i).Name)
			objs[i] = ASTToNode(f.Interface())
		}

		n, err := populateNode(name, objs, false)
		if err != nil {
			panic(fmt.Sprintf("internal error: %s", err))
		}
		return n
	}

	s := reflect.ValueOf(node)
	if s.Kind() == reflect.Slice {
		if s.Len() == 0 {
			return List{}
		}
		if s.Len() == 1 {
			return ASTToNode(s.Index(0).Interface())
		}

		tail := List{}
		for i := s.Len() - 1; i >= 0; i-- {
			head := ASTToNode(s.Index(i).Interface())
			l := List{
				Head: head,
				Tail: tail,
			}
			tail = l
		}
		return tail
	}

	panic(fmt.Sprintf("internal error: unhandled type %T", node))
}
