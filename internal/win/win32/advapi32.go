//go:build windows

package win32

import "unsafe"

var (
	procCredDeleteW = advapi32.NewProc("CredDeleteW")
	procCredFree    = advapi32.NewProc("CredFree")
	procCredReadW   = advapi32.NewProc("CredReadW")
	procCredWriteW  = advapi32.NewProc("CredWriteW")
)

// CredDeleteW deletes a credential.
func CredDeleteW(targetName *uint16, credType uint32, flags uint32) error {
	return boolResult(procCredDeleteW.Call(uintptr(unsafe.Pointer(targetName)), uintptr(credType), uintptr(flags)))
}

// CredFree frees a buffer returned by the credential functions.
func CredFree(buffer unsafe.Pointer) {
	procCredFree.Call(uintptr(buffer))
}

// CredReadW reads a credential; CredFree frees it.
func CredReadW(targetName *uint16, credType uint32, flags uint32) (*CREDENTIALW, error) {
	var out *CREDENTIALW
	if err := boolResult(procCredReadW.Call(uintptr(unsafe.Pointer(targetName)), uintptr(credType), uintptr(flags), uintptr(unsafe.Pointer(&out)))); err != nil {
		return nil, err
	}
	return out, nil
}

// CredWriteW creates or replaces a credential.
func CredWriteW(credential *CREDENTIALW, flags uint32) error {
	return boolResult(procCredWriteW.Call(uintptr(unsafe.Pointer(credential)), uintptr(flags)))
}
