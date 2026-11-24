//go:build windows

package windows

import (
	"unsafe"

	xwin "golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

var (
	clsidDragDropHelper            = com.MustGUID("{4657278A-411B-11D2-839A-00C04FD918D0}")
	iidIDropTargetHelper           = com.MustGUID("{4657278B-411B-11D2-839A-00C04FD918D0}")
	clsidFileOpenDialog            = com.MustGUID("{DC1C5A9C-E88A-4DDE-A5A1-60F82A20AEF7}")
	clsidFileSaveDialog            = com.MustGUID("{C0B4E2F3-BA21-4773-8DBA-335EC946EB8B}")
	iidIFileOpenDialog             = com.MustGUID("{D57C7288-D4AD-4768-BE02-9D969532D960}")
	iidIFileSaveDialog             = com.MustGUID("{84BCCD23-5FDE-4CDB-AEA4-AF64B83D78AB}")
	iidIShellItem                  = com.MustGUID("{43826D1E-E718-42EE-BC55-A1E261C37BFE}")
	shell32Extra                   = xwin.NewLazySystemDLL("shell32.dll")
	procSHGetDesktopFolder         = shell32Extra.NewProc("SHGetDesktopFolder")
	procSHOpenFolderAndSelectItems = shell32Extra.NewProc("SHOpenFolderAndSelectItems")
)

const (
	iFileDialogShow              = com.IUnknownSlots + 0
	iFileDialogSetFileTypes      = com.IUnknownSlots + 1
	iFileDialogSetOptions        = com.IUnknownSlots + 6
	iFileDialogSetFolder         = com.IUnknownSlots + 9
	iFileDialogSetFileName       = com.IUnknownSlots + 12
	iFileDialogSetOkButtonLabel  = com.IUnknownSlots + 15
	iFileDialogGetResult         = com.IUnknownSlots + 17
	iFileOpenDialogGetResults    = com.IUnknownSlots + 24
	iShellItemGetDisplayName     = com.IUnknownSlots + 2
	iShellItemArrayGetCount      = com.IUnknownSlots + 4
	iShellItemArrayGetItemAt     = com.IUnknownSlots + 5
	iShellFolderParseDisplayName = com.IUnknownSlots + 0
)

type filterSpec struct {
	name string
	spec string
}

type comdlgFilterSpec struct {
	pszName *uint16
	pszSpec *uint16
}

type iFileDialog struct{ com.IUnknown }

type iFileOpenDialog struct{ iFileDialog }

func (d *iFileDialog) show(owner HWND) error {
	return com.CallHR(d.Ptr(), iFileDialogShow, uintptr(owner))
}

func (d *iFileDialog) setFileTypes(types []filterSpec) error {
	specs := make([]comdlgFilterSpec, len(types))
	for i, t := range types {
		name, err := xwin.UTF16PtrFromString(t.name)
		if err != nil {
			return err
		}
		spec, err := xwin.UTF16PtrFromString(t.spec)
		if err != nil {
			return err
		}
		specs[i] = comdlgFilterSpec{pszName: name, pszSpec: spec}
	}
	var first *comdlgFilterSpec
	if len(specs) > 0 {
		first = &specs[0]
	}
	return com.CallHR(d.Ptr(), iFileDialogSetFileTypes, uintptr(len(specs)), uintptr(unsafe.Pointer(first)))
}

func (d *iFileDialog) setOptions(options uint32) error {
	return com.CallHR(d.Ptr(), iFileDialogSetOptions, uintptr(options))
}

func (d *iFileDialog) setFolder(item *com.IUnknown) error {
	return com.CallHR(d.Ptr(), iFileDialogSetFolder, uintptr(item.Ptr()))
}

func (d *iFileDialog) setString(slot int, value string) error {
	wide, err := xwin.UTF16PtrFromString(value)
	if err != nil {
		return err
	}
	return com.CallHR(d.Ptr(), slot, uintptr(unsafe.Pointer(wide)))
}

func (d *iFileDialog) setFileName(name string) error {
	return d.setString(iFileDialogSetFileName, name)
}

func (d *iFileDialog) setOkButtonLabel(label string) error {
	return d.setString(iFileDialogSetOkButtonLabel, label)
}

func (d *iFileDialog) getResult() (*iShellItem, error) {
	var out *iShellItem
	if err := com.CallHR(d.Ptr(), iFileDialogGetResult, uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

func (d *iFileOpenDialog) getResults() (*iShellItemArray, error) {
	var out *iShellItemArray
	if err := com.CallHR(d.Ptr(), iFileOpenDialogGetResults, uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

type iShellItem struct{ com.IUnknown }

func (i *iShellItem) getDisplayName(sigdn int32) (string, error) {
	var name *uint16
	if err := com.CallHR(i.Ptr(), iShellItemGetDisplayName, uintptr(uint32(sigdn)), uintptr(unsafe.Pointer(&name))); err != nil {
		return "", err
	}
	return com.TaskMemUTF16(name), nil
}

type iShellItemArray struct{ com.IUnknown }

func (a *iShellItemArray) getCount() (uint32, error) {
	var count uint32
	err := com.CallHR(a.Ptr(), iShellItemArrayGetCount, uintptr(unsafe.Pointer(&count)))
	return count, err
}

func (a *iShellItemArray) getItemAt(index uint32) (*iShellItem, error) {
	var out *iShellItem
	if err := com.CallHR(a.Ptr(), iShellItemArrayGetItemAt, uintptr(index), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

type iShellFolder struct{ com.IUnknown }

func (f *iShellFolder) parseDisplayName(name string) (uintptr, error) {
	wide, err := xwin.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	var pidl uintptr
	err = com.CallHR(f.Ptr(), iShellFolderParseDisplayName, 0, 0, uintptr(unsafe.Pointer(wide)), 0, uintptr(unsafe.Pointer(&pidl)), 0)
	return pidl, err
}

func shGetDesktopFolder() (*iShellFolder, error) {
	var out *iShellFolder
	r, _, _ := procSHGetDesktopFolder.Call(uintptr(unsafe.Pointer(&out)))
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func shOpenFolderAndSelectItems(folder uintptr, items []uintptr, flags uint32) error {
	var first *uintptr
	if len(items) > 0 {
		first = &items[0]
	}
	r, _, _ := procSHOpenFolderAndSelectItems.Call(folder, uintptr(len(items)), uintptr(unsafe.Pointer(first)), uintptr(flags))
	return com.HRESULTFromUintptr(r).Err()
}
