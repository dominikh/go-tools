package astutil

import (
	"go/ast"
	"go/parser"
	"testing"
)

func TestCopyExprTypeFieldsIndependent(t *testing.T) {
	for _, src := range []string{`struct { A, B []int; C struct { D string } }`, `interface { E; F }`} {
		original, err := parser.ParseExpr(src)
		if err != nil {
			t.Fatal(err)
		}
		copied, ok := CopyExpr(original)
		if !ok {
			t.Fatalf("CopyExpr(%s) failed", src)
		}
		var a, b *ast.FieldList
		switch x := original.(type) {
		case *ast.StructType:
			a = x.Fields
			b = copied.(*ast.StructType).Fields
		case *ast.InterfaceType:
			a = x.Methods
			b = copied.(*ast.InterfaceType).Methods
		}
		if a == b || a.List[0] == b.List[0] {
			t.Fatalf("%s shares field storage", src)
		}
		if len(a.List[0].Names) > 0 {
			b.List[0].Names[0].Name = "Changed"
			if a.List[0].Names[0].Name == "Changed" {
				t.Fatal("shared names")
			}
		}
		b.List[0].Type = ast.NewIdent("Changed")
		if id, ok := a.List[0].Type.(*ast.Ident); ok && id.Name == "Changed" {
			t.Fatal("shared type")
		}
	}
}

func TestCopyExprNestedFieldsAndMetadata(t *testing.T) {
	original := &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{{Doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "// doc"}}}, Comment: &ast.CommentGroup{List: []*ast.Comment{{Text: "// tail"}}}, Tag: &ast.BasicLit{Value: "tag"}, Type: &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("int")}}}}}}}}
	node, ok := CopyExpr(original)
	if !ok {
		t.Fatal("copy failed")
	}
	field := node.(*ast.StructType).Fields.List[0]
	field.Doc.List[0].Text = "changed"
	field.Comment.List[0].Text = "changed"
	field.Tag.Value = "changed"
	field.Type.(*ast.StructType).Fields.List[0].Type.(*ast.Ident).Name = "changed"
	source := original.Fields.List[0]
	if source.Doc.List[0].Text != "// doc" || source.Comment.List[0].Text != "// tail" || source.Tag.Value != "tag" || source.Type.(*ast.StructType).Fields.List[0].Type.(*ast.Ident).Name != "int" {
		t.Fatal("copy mutated source")
	}
}
func TestCopyExprInterfaceMethod(t *testing.T) {
	x, err := parser.ParseExpr("interface { M(a int) string }")
	if err != nil {
		t.Fatal(err)
	}
	c, ok := CopyExpr(x)
	if !ok {
		t.Fatal("method copy failed")
	}
	method := c.(*ast.InterfaceType).Methods.List[0].Type.(*ast.FuncType)
	method.Params.List[0].Names[0].Name = "changed"
	method.Results.List[0].Type.(*ast.Ident).Name = "changed"
	source := x.(*ast.InterfaceType).Methods.List[0].Type.(*ast.FuncType)
	if source.Params.List[0].Names[0].Name != "a" || source.Results.List[0].Type.(*ast.Ident).Name != "string" {
		t.Fatal("method copy mutated source")
	}
}
