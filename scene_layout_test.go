package gpui

import (
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unsafe"
)

type gpuLayoutField struct {
	name   string
	offset uintptr
}

type gpuLayout struct {
	goType reflect.Type
	size   uintptr
	shader string
	fields []gpuLayoutField
}

var gpuLayouts = []gpuLayout{
	{reflect.TypeFor[Hsla](), unsafe.Sizeof(Hsla{}), "Hsla", []gpuLayoutField{
		{"h", 0}, {"s", 4}, {"l", 8}, {"a", 12},
	}},
	{reflect.TypeFor[LinearColorStop](), unsafe.Sizeof(LinearColorStop{}), "LinearColorStop", []gpuLayoutField{
		{"color", 0}, {"percentage", 16},
	}},
	{reflect.TypeFor[Background](), unsafe.Sizeof(Background{}), "Background", []gpuLayoutField{
		{"tag", 0}, {"color_space", 4}, {"solid", 8}, {"gradient_angle_or_pattern_height", 24}, {"colors", 28}, {"pad", 68},
	}},
	{reflect.TypeFor[Bounds[ScaledPixels]](), unsafe.Sizeof(Bounds[ScaledPixels]{}), "Bounds", []gpuLayoutField{
		{"origin", 0}, {"size", 8},
	}},
	{reflect.TypeFor[Corners[ScaledPixels]](), unsafe.Sizeof(Corners[ScaledPixels]{}), "Corners", []gpuLayoutField{
		{"top_left", 0}, {"top_right", 4}, {"bottom_right", 8}, {"bottom_left", 12},
	}},
	{reflect.TypeFor[Edges[ScaledPixels]](), unsafe.Sizeof(Edges[ScaledPixels]{}), "Edges", []gpuLayoutField{
		{"top", 0}, {"right", 4}, {"bottom", 8}, {"left", 12},
	}},
	{reflect.TypeFor[AtlasTextureID](), unsafe.Sizeof(AtlasTextureID{}), "AtlasTextureId", []gpuLayoutField{
		{"index", 0}, {"kind", 4},
	}},
	{reflect.TypeFor[AtlasTile](), unsafe.Sizeof(AtlasTile{}), "AtlasTile", []gpuLayoutField{
		{"texture_id", 0}, {"tile_id", 8}, {"padding", 12}, {"bounds", 16},
	}},
	{reflect.TypeFor[TransformationMatrix](), unsafe.Sizeof(TransformationMatrix{}), "TransformationMatrix", []gpuLayoutField{
		{"rotation_scale", 0}, {"translation", 16},
	}},
	{reflect.TypeFor[Quad](), unsafe.Sizeof(Quad{}), "Quad", []gpuLayoutField{
		{"order", 0}, {"border_style", 4}, {"bounds", 8}, {"content_mask", 24}, {"background", 40},
		{"border_color", 112}, {"corner_radii", 128}, {"border_widths", 144},
	}},
	{reflect.TypeFor[Shadow](), unsafe.Sizeof(Shadow{}), "Shadow", []gpuLayoutField{
		{"order", 0}, {"blur_radius", 4}, {"bounds", 8}, {"corner_radii", 24}, {"content_mask", 40},
		{"color", 56}, {"shape_bounds", 72}, {"shape_corner_radii", 88}, {"inset", 104}, {"pad", 108},
	}},
	{reflect.TypeFor[Underline](), unsafe.Sizeof(Underline{}), "Underline", []gpuLayoutField{
		{"order", 0}, {"pad", 4}, {"bounds", 8}, {"content_mask", 24}, {"color", 40}, {"thickness", 56}, {"wavy", 60},
	}},
	{reflect.TypeFor[MonochromeSprite](), unsafe.Sizeof(MonochromeSprite{}), "MonochromeSprite", []gpuLayoutField{
		{"order", 0}, {"pad", 4}, {"tile", 8}, {"bounds", 40}, {"content_mask", 56}, {"color", 72}, {"transformation", 88},
	}},
	{reflect.TypeFor[SubpixelSprite](), unsafe.Sizeof(SubpixelSprite{}), "MonochromeSprite", []gpuLayoutField{
		{"order", 0}, {"pad", 4}, {"tile", 8}, {"bounds", 40}, {"content_mask", 56}, {"color", 72}, {"transformation", 88},
	}},
	{reflect.TypeFor[PolychromeSprite](), unsafe.Sizeof(PolychromeSprite{}), "PolychromeSprite", []gpuLayoutField{
		{"order", 0}, {"pad", 4}, {"tile", 8}, {"grayscale", 40}, {"opacity", 44}, {"bounds", 48},
		{"content_mask", 64}, {"corner_radii", 80},
	}},
	{reflect.TypeFor[PathCurve](), unsafe.Sizeof(PathCurve{}), "PathCurve", []gpuLayoutField{
		{"p0", 0}, {"p1", 8}, {"a", 16}, {"b", 24}, {"sign", 32}, {"pad", 36},
	}},
	{reflect.TypeFor[PathPaint](), unsafe.Sizeof(PathPaint{}), "PathPaint", []gpuLayoutField{
		{"bounds", 0}, {"content_mask", 16}, {"color", 32}, {"even_odd", 104}, {"pad", 108},
	}},
	{reflect.TypeFor[PathTile](), unsafe.Sizeof(PathTile{}), "PathTile", []gpuLayoutField{
		{"paint", 0}, {"curve_start", 4}, {"curve_count", 8}, {"backdrop", 12}, {"top_left", 16},
		{"bottom_right", 24}, {"reference", 32}, {"curves_top_left", 40}, {"curves_bottom_right", 48},
	}},
}

var gpuLayoutSizes = map[string]uintptr{
	"Hsla": 16, "LinearColorStop": 20, "Background": 72, "Bounds": 16, "Corners": 16, "Edges": 16,
	"AtlasTextureId": 8, "AtlasTile": 32, "TransformationMatrix": 24, "Quad": 160, "Shadow": 112,
	"Underline": 64, "MonochromeSprite": 112, "PolychromeSprite": 96, "PathCurve": 40, "PathPaint": 112,
	"PathTile": 56,
}

var gpuLayoutGoNames = map[string]string{
	"MonochromeSprite.Animation": "pad",
	"Background.ColorSpace":      "color_space",
	"AtlasTile.TextureID":        "texture_id",
	"AtlasTile.TileID":           "tile_id",
}

func snakeCase(name string) string {
	var b strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func goTypeName(t reflect.Type) string {
	name := t.Name()
	if i := strings.IndexByte(name, '['); i >= 0 {
		name = name[:i]
	}
	return name
}

func TestSceneLayout_GoStructsMatchShaderLayouts(t *testing.T) {
	for _, layout := range gpuLayouts {
		name := goTypeName(layout.goType)
		if want := gpuLayoutSizes[layout.shader]; layout.size != want {
			t.Errorf("unsafe.Sizeof(%s) = %d, want %d", name, layout.size, want)
		}
		if layout.goType.NumField() != len(layout.fields) {
			t.Errorf("%s has %d fields, want %d", name, layout.goType.NumField(), len(layout.fields))
			continue
		}
		for i, want := range layout.fields {
			field := layout.goType.Field(i)
			shaderName, ok := gpuLayoutGoNames[name+"."+field.Name]
			if !ok {
				shaderName = snakeCase(field.Name)
			}
			if shaderName != want.name {
				t.Errorf("%s field %d is %s (%s), want %s", name, i, field.Name, shaderName, want.name)
			}
			if field.Offset != want.offset {
				t.Errorf("offset of %s.%s = %d, want %d", name, field.Name, field.Offset, want.offset)
			}
			if field.Type.Align() > 4 {
				t.Errorf("%s.%s has alignment %d; GPU fields must be 4-byte aligned", name, field.Name, field.Type.Align())
			}
		}
	}
}

type shaderField struct {
	name  string
	typ   string
	count int
}

func parseShaderStructs(t *testing.T, path string, field *regexp.Regexp, decode func([]string) shaderField) map[string][]shaderField {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("reference shader not available: %v", err)
	}
	structs := map[string][]shaderField{}
	header := regexp.MustCompile(`^struct (\w+) \{`)
	var current string
	for _, line := range strings.Split(string(src), "\n") {
		if m := header.FindStringSubmatch(line); m != nil {
			current = m[1]
			structs[current] = nil
			continue
		}
		if current == "" {
			continue
		}
		if strings.HasPrefix(line, "}") {
			current = ""
			continue
		}
		code := line
		if i := strings.Index(code, "//"); i >= 0 {
			code = code[:i]
		}
		if m := field.FindStringSubmatch(code); m != nil {
			structs[current] = append(structs[current], decode(m))
		}
	}
	return structs
}

type shaderLayoutRules struct {
	scalar func(typ string) (size, align uintptr, ok bool)
}

func shaderStructLayout(t *testing.T, structs map[string][]shaderField, rules shaderLayoutRules, name string) (size, align uintptr, offsets []uintptr) {
	t.Helper()
	fields, ok := structs[name]
	if !ok {
		t.Fatalf("struct %s not found", name)
	}
	align = 1
	var offset uintptr
	for _, field := range fields {
		fieldSize, fieldAlign, ok := rules.scalar(field.typ)
		if !ok {
			fieldSize, fieldAlign, _ = shaderStructLayout(t, structs, rules, field.typ)
		}
		if field.count > 0 {
			stride := (fieldSize + fieldAlign - 1) / fieldAlign * fieldAlign
			fieldSize = stride * uintptr(field.count)
		}
		offset = (offset + fieldAlign - 1) / fieldAlign * fieldAlign
		offsets = append(offsets, offset)
		offset += fieldSize
		align = max(align, fieldAlign)
	}
	size = (offset + align - 1) / align * align
	return size, align, offsets
}

func checkShaderLayouts(t *testing.T, structs map[string][]shaderField, rules shaderLayoutRules) {
	t.Helper()
	for _, layout := range gpuLayouts {
		fields, ok := structs[layout.shader]
		if !ok {
			t.Errorf("shader struct %s not found", layout.shader)
			continue
		}
		size, _, offsets := shaderStructLayout(t, structs, rules, layout.shader)
		if size != layout.size {
			t.Errorf("shader %s is %d bytes, Go %s is %d", layout.shader, size, goTypeName(layout.goType), layout.size)
		}
		if len(fields) != len(layout.fields) {
			t.Errorf("shader %s has %d fields, want %d", layout.shader, len(fields), len(layout.fields))
			continue
		}
		for i, want := range layout.fields {
			if fields[i].name != want.name || offsets[i] != want.offset {
				t.Errorf("shader %s field %d is %s at %d, want %s at %d", layout.shader, i, fields[i].name, offsets[i], want.name, want.offset)
			}
		}
	}
}

func TestSceneLayout_WGSL(t *testing.T) {
	field := regexp.MustCompile(`^\s*(\w+)\s*:\s*([\w<>, ]+?)\s*,?\s*$`)
	array := regexp.MustCompile(`^array<(\w+),\s*(\d+)>$`)
	structs := parseShaderStructs(t, "_ref/gpui_wgpu/src/shaders.wgsl", field, func(m []string) shaderField {
		typ := m[2]
		if a := array.FindStringSubmatch(typ); a != nil {
			n, _ := strconv.Atoi(a[2])
			return shaderField{name: m[1], typ: a[1], count: n}
		}
		return shaderField{name: m[1], typ: typ}
	})
	rules := shaderLayoutRules{scalar: func(typ string) (uintptr, uintptr, bool) {
		switch typ {
		case "f32", "u32", "i32":
			return 4, 4, true
		case "vec2<f32>", "vec2<u32>", "vec2<i32>":
			return 8, 8, true
		case "vec4<f32>", "vec4<u32>", "vec4<i32>":
			return 16, 16, true
		case "mat2x2<f32>":
			return 16, 8, true
		}
		return 0, 0, false
	}}
	checkShaderLayouts(t, structs, rules)
}

func TestSceneLayout_HLSL(t *testing.T) {
	field := regexp.MustCompile(`^\s*(\w+)\s+(\w+)(?:\[(\d+)\])?\s*;\s*$`)
	structs := parseShaderStructs(t, "_ref/gpui_windows/src/shaders.hlsl", field, func(m []string) shaderField {
		n := 0
		if m[3] != "" {
			n, _ = strconv.Atoi(m[3])
		}
		return shaderField{name: m[2], typ: m[1], count: n}
	})
	rules := shaderLayoutRules{scalar: func(typ string) (uintptr, uintptr, bool) {
		switch typ {
		case "float", "uint", "int":
			return 4, 4, true
		case "float2", "uint2", "int2":
			return 8, 4, true
		case "float4", "uint4", "int4":
			return 16, 4, true
		case "float2x2":
			return 16, 4, true
		}
		return 0, 0, false
	}}
	checkShaderLayouts(t, structs, rules)
}

func TestSceneLayout_Metal(t *testing.T) {
	src, err := os.ReadFile("_ref/gpui_apple/src/shaders.metal")
	if err != nil {
		t.Skipf("reference shader not available: %v", err)
	}
	for _, name := range []string{"Quad", "Shadow", "Underline", "MonochromeSprite", "PolychromeSprite", "PathTile", "PathCurve", "PathPaint"} {
		if !regexp.MustCompile(`constant ` + name + ` \*`).Match(src) {
			t.Errorf("shaders.metal does not read a %s buffer", name)
		}
	}
	for _, layout := range gpuLayouts {
		var offset uintptr
		for i := 0; i < layout.goType.NumField(); i++ {
			field := layout.goType.Field(i)
			if field.Offset != offset {
				t.Errorf("%s.%s is at %d, but cbindgen's C struct packs it at %d", goTypeName(layout.goType), field.Name, field.Offset, offset)
			}
			offset += field.Type.Size()
		}
		if offset != layout.size {
			t.Errorf("%s fields end at %d, size %d", goTypeName(layout.goType), offset, layout.size)
		}
	}
}
