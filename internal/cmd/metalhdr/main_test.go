package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	gpui "github.com/megascan/gpui-go"
)

const (
	refDir  = "../../../_ref"
	genFile = "../../../platform/apple/shaders_scene_gen.go"
)

func TestMetalhdr_GeneratedFileIsCurrent(t *testing.T) {
	header, err := generate("", refDir)
	if errors.Is(err, errSkip) {
		t.Skipf("cannot regenerate: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	want, err := render(header)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(genFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale; run go generate in platform/apple", genFile)
	}
}

func TestMetalhdr_ExportListMatchesBuildRs(t *testing.T) {
	data, err := os.ReadFile(refDir + "/gpui_apple/build.rs")
	if err != nil {
		t.Skipf("_ref not available: %v", err)
	}
	src := string(data)
	start := strings.Index(src, "config.export.include.extend([")
	end := strings.Index(src[start:], "]);")
	if start < 0 || end < 0 {
		t.Fatal("export include list not found in build.rs")
	}
	var names []string
	for _, m := range regexp.MustCompile(`"(\w+)"\.into\(\)`).FindAllStringSubmatch(src[start:start+end], -1) {
		names = append(names, m[1])
	}
	if !reflect.DeepEqual(names, exportInclude) {
		t.Fatalf("build.rs exports %v, metalhdr exports %v", names, exportInclude)
	}
}

type cField struct {
	typ  string
	name string
	dims []int
}

type cHeader struct {
	scalars map[string]string
	structs map[string][]cField
	order   []string
	enums   map[string]map[string]int64
	opaque  map[string]bool
}

func committedHeader(t *testing.T) string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), genFile, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			if len(vs.Names) == 1 && vs.Names[0].Name == "sceneHeader" {
				header, err := strconv.Unquote(vs.Values[0].(*ast.BasicLit).Value)
				if err != nil {
					t.Fatal(err)
				}
				return header
			}
		}
	}
	t.Fatal("sceneHeader not found")
	return ""
}

func parseHeader(t *testing.T, src string) cHeader {
	t.Helper()
	src = regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(src, "")
	src = regexp.MustCompile(`(?m)^#.*$`).ReplaceAllString(src, "")
	h := cHeader{
		scalars: map[string]string{},
		structs: map[string][]cField{},
		enums:   map[string]map[string]int64{},
		opaque:  map[string]bool{},
	}
	decl := regexp.MustCompile(`(?s)typedef\s+(enum|struct)\s+(\w+)\s*\{(.*?)\}\s*(\w+)\s*;|typedef\s+struct\s+(\w+)\s+(\w+)\s*;|typedef\s+(\w+)\s+(\w+)\s*;`)
	rest := decl.ReplaceAllStringFunc(src, func(m string) string {
		g := decl.FindStringSubmatch(m)
		switch {
		case g[1] == "enum":
			values := map[string]int64{}
			for item := range strings.SplitSeq(g[3], ",") {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				name, value, ok := strings.Cut(item, "=")
				if !ok {
					t.Fatalf("enum %s item %q has no value", g[2], item)
				}
				v, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
				if err != nil {
					t.Fatal(err)
				}
				values[strings.TrimSpace(name)] = v
			}
			h.enums[g[2]] = values
		case g[1] == "struct":
			var fields []cField
			field := regexp.MustCompile(`^(?:struct\s+|enum\s+)?(\w+)\s+(\w+)((?:\[\d+\])*)$`)
			for line := range strings.SplitSeq(g[3], ";") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				f := field.FindStringSubmatch(line)
				if f == nil {
					t.Fatalf("struct %s: cannot parse field %q", g[2], line)
				}
				var dims []int
				for _, d := range regexp.MustCompile(`\d+`).FindAllString(f[3], -1) {
					n, _ := strconv.Atoi(d)
					dims = append(dims, n)
				}
				fields = append(fields, cField{typ: f[1], name: f[2], dims: dims})
			}
			h.structs[g[2]] = fields
			h.order = append(h.order, g[2])
		case g[5] != "":
			h.opaque[g[5]] = true
		default:
			h.scalars[g[8]] = g[7]
		}
		return ""
	})
	if strings.TrimSpace(rest) != "" {
		t.Fatalf("unparsed header text: %q", strings.TrimSpace(rest))
	}
	return h
}

func (h cHeader) size(t *testing.T, typ string) uintptr {
	t.Helper()
	switch typ {
	case "float", "int32_t", "uint32_t":
		return 4
	}
	if base, ok := h.scalars[typ]; ok {
		return h.size(t, base)
	}
	if _, ok := h.enums[typ]; ok {
		return 4
	}
	if fields, ok := h.structs[typ]; ok {
		var size uintptr
		for _, f := range fields {
			size += h.fieldSize(t, f)
		}
		return size
	}
	t.Fatalf("unknown C type %s", typ)
	return 0
}

func (h cHeader) fieldSize(t *testing.T, f cField) uintptr {
	size := h.size(t, f.typ)
	for _, d := range f.dims {
		size *= uintptr(d)
	}
	return size
}

type surfaceBounds struct {
	Bounds      gpui.Bounds[gpui.ScaledPixels]
	ContentMask gpui.ContentMask[gpui.ScaledPixels]
}

var goStructs = map[string]reflect.Type{
	"Hsla":                     reflect.TypeFor[gpui.Hsla](),
	"AtlasTextureId":           reflect.TypeFor[gpui.AtlasTextureID](),
	"TileId":                   reflect.TypeFor[gpui.TileID](),
	"Point_DevicePixels":       reflect.TypeFor[gpui.Point[gpui.DevicePixels]](),
	"Size_DevicePixels":        reflect.TypeFor[gpui.Size[gpui.DevicePixels]](),
	"Bounds_DevicePixels":      reflect.TypeFor[gpui.Bounds[gpui.DevicePixels]](),
	"AtlasTile":                reflect.TypeFor[gpui.AtlasTile](),
	"Point_f32":                reflect.TypeFor[gpui.Point[float32]](),
	"PathTile":                 reflect.TypeFor[gpui.PathTile](),
	"PathCurve":                reflect.TypeFor[gpui.PathCurve](),
	"Point_ScaledPixels":       reflect.TypeFor[gpui.Point[gpui.ScaledPixels]](),
	"Size_ScaledPixels":        reflect.TypeFor[gpui.Size[gpui.ScaledPixels]](),
	"Bounds_ScaledPixels":      reflect.TypeFor[gpui.Bounds[gpui.ScaledPixels]](),
	"ContentMask_ScaledPixels": reflect.TypeFor[gpui.ContentMask[gpui.ScaledPixels]](),
	"LinearColorStop":          reflect.TypeFor[gpui.LinearColorStop](),
	"Background":               reflect.TypeFor[gpui.Background](),
	"PathPaint":                reflect.TypeFor[gpui.PathPaint](),
	"Corners_ScaledPixels":     reflect.TypeFor[gpui.Corners[gpui.ScaledPixels]](),
	"Shadow":                   reflect.TypeFor[gpui.Shadow](),
	"Underline":                reflect.TypeFor[gpui.Underline](),
	"Edges_ScaledPixels":       reflect.TypeFor[gpui.Edges[gpui.ScaledPixels]](),
	"Quad":                     reflect.TypeFor[gpui.Quad](),
	"TransformationMatrix":     reflect.TypeFor[gpui.TransformationMatrix](),
	"MonochromeSprite":         reflect.TypeFor[gpui.MonochromeSprite](),
	"PolychromeSprite":         reflect.TypeFor[gpui.PolychromeSprite](),
	"SurfaceBounds":            reflect.TypeFor[surfaceBounds](),
}

var goScalars = map[string]reflect.Type{
	"Pixels":            reflect.TypeFor[gpui.Pixels](),
	"DevicePixels":      reflect.TypeFor[gpui.DevicePixels](),
	"ScaledPixels":      reflect.TypeFor[gpui.ScaledPixels](),
	"PaddedBool32":      reflect.TypeFor[gpui.PaddedBool32](),
	"DrawOrder":         reflect.TypeFor[gpui.DrawOrder](),
	"SpriteAnimationId": reflect.TypeFor[gpui.SpriteAnimationID](),
}

var goEnums = map[string]map[string]int64{
	"AtlasTextureKind": {
		"AtlasTextureKind_Monochrome": int64(gpui.AtlasTextureKindMonochrome),
		"AtlasTextureKind_Polychrome": int64(gpui.AtlasTextureKindPolychrome),
		"AtlasTextureKind_Subpixel":   int64(gpui.AtlasTextureKindSubpixel),
	},
	"BackgroundTag": {
		"BackgroundTag_Solid":          int64(gpui.BackgroundTagSolid),
		"BackgroundTag_LinearGradient": int64(gpui.BackgroundTagLinearGradient),
		"BackgroundTag_PatternSlash":   int64(gpui.BackgroundTagPatternSlash),
		"BackgroundTag_Checkerboard":   int64(gpui.BackgroundTagCheckerboard),
	},
	"BorderStyle": {
		"BorderStyle_Solid":  int64(gpui.BorderStyleSolid),
		"BorderStyle_Dashed": int64(gpui.BorderStyleDashed),
	},
	"ColorSpace": {
		"ColorSpace_Srgb":  int64(gpui.ColorSpaceSrgb),
		"ColorSpace_Oklab": int64(gpui.ColorSpaceOklab),
	},
}

var rendererEnums = map[string]bool{
	"GlobalInputIndex":    true,
	"PathTileInputIndex":  true,
	"QuadInputIndex":      true,
	"ShadowInputIndex":    true,
	"SpriteInputIndex":    true,
	"SurfaceInputIndex":   true,
	"UnderlineInputIndex": true,
}

func normalize(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", ""))
}

func TestMetalhdr_SceneHeaderMatchesGoLayout(t *testing.T) {
	h := parseHeader(t, committedHeader(t))
	if len(h.structs) == 0 {
		t.Fatal("header declares no structs")
	}
	for name, base := range h.scalars {
		goType, ok := goScalars[name]
		if !ok {
			t.Errorf("scalar typedef %s has no Go type", name)
			continue
		}
		if goType.Size() != h.size(t, base) {
			t.Errorf("sizeof(%s) = %d, Go %s is %d", name, h.size(t, base), goType, goType.Size())
		}
		isFloat := base == "float"
		if isFloat != (goType.Kind() == reflect.Float32) {
			t.Errorf("%s is %s in C, Go %s is %s", name, base, goType, goType.Kind())
		}
	}
	for name, values := range h.enums {
		if rendererEnums[name] {
			continue
		}
		want, ok := goEnums[name]
		if !ok {
			t.Errorf("enum %s has no Go constants", name)
			continue
		}
		if !reflect.DeepEqual(values, want) {
			t.Errorf("enum %s = %v, Go = %v", name, values, want)
		}
	}
	for _, name := range h.order {
		fields := h.structs[name]
		goType, ok := goStructs[name]
		if !ok {
			t.Errorf("struct %s has no Go type", name)
			continue
		}
		cSize := h.size(t, name)
		if goType.Size() != cSize {
			t.Errorf("sizeof(%s) = %d, Go %s is %d", name, cSize, goType, goType.Size())
		}
		if goType.Kind() != reflect.Struct {
			if len(fields) != 1 || fields[0].name != "_0" {
				t.Errorf("struct %s maps to non-struct Go %s", name, goType)
			}
			continue
		}
		if goType.NumField() != len(fields) {
			t.Errorf("struct %s has %d fields, Go %s has %d", name, len(fields), goType, goType.NumField())
			continue
		}
		var offset uintptr
		for i, f := range fields {
			goField := goType.Field(i)
			if normalize(goField.Name) != normalize(f.name) {
				t.Errorf("%s field %d is %s, Go field is %s", name, i, f.name, goField.Name)
			}
			if goField.Offset != offset {
				t.Errorf("offsetof(%s, %s) = %d, Go %s.%s is at %d", name, f.name, offset, goType, goField.Name, goField.Offset)
			}
			size := h.fieldSize(t, f)
			if goField.Type.Size() != size {
				t.Errorf("sizeof(%s.%s) = %d, Go %s.%s is %d", name, f.name, size, goType, goField.Name, goField.Type.Size())
			}
			offset += size
		}
	}
	for name := range goStructs {
		if _, ok := h.structs[name]; !ok {
			t.Errorf("Go mapping %s has no struct in the header", name)
		}
	}
}
