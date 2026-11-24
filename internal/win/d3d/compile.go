//go:build windows

package d3d

import (
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

var (
	modD3DCompiler = windows.NewLazySystemDLL("d3dcompiler_47.dll")
	procD3DCompile = modD3DCompiler.NewProc("D3DCompile")
)

// ID3DBlob is a buffer returned by the compiler.
type ID3DBlob struct{ com.IUnknown }

// GetBufferPointer returns the blob's data.
func (b *ID3DBlob) GetBufferPointer() unsafe.Pointer {
	return com.PtrFromUintptr(com.Call(b.Ptr(), com.IUnknownSlots))
}

// GetBufferSize returns the blob's size in bytes.
func (b *ID3DBlob) GetBufferSize() uintptr {
	return com.Call(b.Ptr(), com.IUnknownSlots+1)
}

// Bytes returns the blob's data; it is valid until the blob is released.
func (b *ID3DBlob) Bytes() []byte {
	n := b.GetBufferSize()
	if n == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(b.GetBufferPointer()), n)
}

// String returns the blob's data as a NUL-terminated string, as compiler messages are.
func (b *ID3DBlob) String() string {
	data := b.Bytes()
	for i, c := range data {
		if c == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

// D3DCompile compiles HLSL source. include may be nil. On failure the error blob, when the
// compiler produced one, is returned alongside the error; the caller releases both blobs.
func D3DCompile(source []byte, sourceName string, include *Include, entryPoint, target string, flags1, flags2 uint32) (*ID3DBlob, *ID3DBlob, error) {
	if err := procD3DCompile.Find(); err != nil {
		return nil, nil, err
	}
	name, err := windows.BytePtrFromString(sourceName)
	if err != nil {
		return nil, nil, err
	}
	entry, err := windows.BytePtrFromString(entryPoint)
	if err != nil {
		return nil, nil, err
	}
	tgt, err := windows.BytePtrFromString(target)
	if err != nil {
		return nil, nil, err
	}
	var includePtr uintptr
	if include != nil {
		includePtr = uintptr(include.obj)
	}
	var code, errorMsgs *ID3DBlob
	r, _, _ := procD3DCompile.Call(
		uintptr(slicePtr(source)),
		uintptr(len(source)),
		uintptr(unsafe.Pointer(name)),
		0,
		includePtr,
		uintptr(unsafe.Pointer(entry)),
		uintptr(unsafe.Pointer(tgt)),
		uintptr(flags1),
		uintptr(flags2),
		uintptr(unsafe.Pointer(&code)),
		uintptr(unsafe.Pointer(&errorMsgs)),
	)
	runtime.KeepAlive(source)
	runtime.KeepAlive(name)
	runtime.KeepAlive(entry)
	runtime.KeepAlive(tgt)
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		if code != nil {
			code.Release()
		}
		return nil, errorMsgs, err
	}
	return code, errorMsgs, nil
}

// Include is an ID3DInclude implemented in Go that serves a fixed set of files by name. It is
// not a COM object: its vtable holds only Open and Close. Open fails for any name it does not
// serve; Close frees nothing, because the served data lives as long as the Include.
type Include struct {
	obj   unsafe.Pointer
	files map[string]includeFile
}

type includeFile struct {
	data unsafe.Pointer
	size uint32
}

var (
	includes      sync.Map
	includeVtable = sync.OnceValue(func() unsafe.Pointer {
		table := com.CoTaskMemAlloc(2 * unsafe.Sizeof(uintptr(0)))
		if table == nil {
			panic("d3d: out of memory allocating the ID3DInclude vtable")
		}
		slots := unsafe.Slice((*uintptr)(table), 2)
		slots[0] = windows.NewCallback(includeOpen)
		slots[1] = windows.NewCallback(includeClose)
		return table
	})
)

// NewInclude returns an Include serving files, whose contents are copied to native memory.
func NewInclude(files map[string][]byte) *Include {
	obj := com.CoTaskMemAlloc(unsafe.Sizeof(uintptr(0)))
	if obj == nil {
		panic("d3d: out of memory allocating an ID3DInclude")
	}
	*(*unsafe.Pointer)(obj) = includeVtable()
	inc := &Include{obj: obj, files: make(map[string]includeFile, len(files))}
	for name, contents := range files {
		size := len(contents)
		alloc := size
		if alloc == 0 {
			alloc = 1
		}
		data := com.CoTaskMemAlloc(uintptr(alloc))
		if data == nil {
			panic("d3d: out of memory copying an include file")
		}
		copy(unsafe.Slice((*byte)(data), alloc), contents)
		inc.files[name] = includeFile{data: data, size: uint32(size)}
	}
	includes.Store(uintptr(obj), inc)
	return inc
}

// Free releases the Include's native memory. It must not be used after.
func (i *Include) Free() {
	if i.obj == nil {
		return
	}
	includes.Delete(uintptr(i.obj))
	for _, f := range i.files {
		com.CoTaskMemFree(f.data)
	}
	com.CoTaskMemFree(i.obj)
	i.obj = nil
	i.files = nil
}

func includeOpen(this, includeType, fileName, parentData, ppData, pBytes uintptr) uintptr {
	v, ok := includes.Load(this)
	if !ok {
		return hresultRet(com.E_FAIL)
	}
	inc := v.(*Include)
	name := windows.BytePtrToString((*byte)(com.PtrFromUintptr(fileName)))
	f, ok := inc.files[name]
	if !ok {
		return hresultRet(com.HRESULTFromWin32(uint32(windows.ERROR_FILE_NOT_FOUND)))
	}
	*(*uintptr)(com.PtrFromUintptr(ppData)) = uintptr(f.data)
	*(*uint32)(com.PtrFromUintptr(pBytes)) = f.size
	return 0
}

func includeClose(this, data uintptr) uintptr {
	return 0
}

func hresultRet(h com.HRESULT) uintptr {
	return uintptr(uint32(h))
}
