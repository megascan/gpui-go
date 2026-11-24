package gpui

import (
	"embed"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"sync"
)

//go:embed styled.go styled_gen.go
var styledReflectionSources embed.FS

// InspectorReflectionMethods returns the methods of T with the shape func(T) T, the Go form of
// derive_inspector_reflection's methods. Only methods named in trait's method set are included
// when trait is non-nil; docs supplies each method's documentation by name.
func InspectorReflectionMethods[T any](trait reflect.Type, docs map[string]string) []FunctionReflection[T] {
	typ := reflect.TypeFor[T]()
	var allowed map[string]bool
	if trait != nil {
		allowed = make(map[string]bool, trait.NumMethod())
		for method := range trait.Methods() {
			allowed[method.Name] = true
		}
	}
	var methods []FunctionReflection[T]
	for method := range typ.Methods() {
		if allowed != nil && !allowed[method.Name] {
			continue
		}
		fnType := method.Type
		if fnType.NumIn() != 1 || fnType.NumOut() != 1 || fnType.Out(0) != typ {
			continue
		}
		function := method.Func
		documentation := None[string]()
		if doc, ok := docs[method.Name]; ok {
			documentation = Some(doc)
		}
		methods = append(methods, FunctionReflection[T]{
			Name: method.Name,
			Function: func(value any) any {
				concrete, ok := value.(T)
				if !ok {
					panic("Type mismatch in reflection wrapper")
				}
				return function.Call([]reflect.Value{reflect.ValueOf(&concrete).Elem()})[0].Interface()
			},
			Documentation: documentation,
		})
	}
	return methods
}

// InspectorReflectionFindMethod returns the method of InspectorReflectionMethods named name.
func InspectorReflectionFindMethod[T any](trait reflect.Type, docs map[string]string, name string) (FunctionReflection[T], bool) {
	for _, method := range InspectorReflectionMethods[T](trait, docs) {
		if method.Name == name {
			return method, true
		}
	}
	return FunctionReflection[T]{}, false
}

// StyledReflectionMethods returns the Styled methods of S with the shape func(S) S, with their
// documentation (styled_reflection::methods).
func StyledReflectionMethods[S Styler]() []FunctionReflection[S] {
	return InspectorReflectionMethods[S](reflect.TypeFor[*Styled[S]](), styledMethodDocs())
}

// StyledReflectionFindMethod returns the Styled method of S named name (styled_reflection::find_method).
func StyledReflectionFindMethod[S Styler](name string) (FunctionReflection[S], bool) {
	return InspectorReflectionFindMethod[S](reflect.TypeFor[*Styled[S]](), styledMethodDocs(), name)
}

var styledMethodDocs = sync.OnceValue(func() map[string]string {
	docs := make(map[string]string)
	fset := token.NewFileSet()
	for _, name := range []string{"styled.go", "styled_gen.go"} {
		src, err := styledReflectionSources.ReadFile(name)
		if err != nil {
			panic(err)
		}
		file, err := parser.ParseFile(fset, name, src, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			panic(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Doc == nil || !isStyledReceiver(fn.Recv) {
				continue
			}
			docs[fn.Name.Name] = strings.TrimSpace(fn.Doc.Text())
		}
	}
	return docs
})

func isStyledReceiver(recv *ast.FieldList) bool {
	if len(recv.List) != 1 {
		return false
	}
	expr := recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if index, ok := expr.(*ast.IndexExpr); ok {
		expr = index.X
	}
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "Styled"
}
