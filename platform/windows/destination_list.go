//go:build windows

package windows

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf16"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/com"
)

// JumpList is the taskbar jump list: the dock menu tasks and the recent workspaces.
type JumpList struct {
	dockMenus        []DockMenuItem
	recentWorkspaces [][]string
}

// NewJumpList returns an empty jump list.
func NewJumpList() JumpList {
	return JumpList{}
}

// DockMenuItem is one task of the jump list.
type DockMenuItem struct {
	name        gpui.SharedString
	description gpui.SharedString
	action      gpui.Action
}

// NewDockMenuItem converts a MenuItemAction into a jump list task.
func NewDockMenuItem(item gpui.MenuItem) (DockMenuItem, error) {
	switch item := item.(type) {
	case gpui.MenuItemAction:
		description := item.Name
		if item.Name == "New Window" {
			description = "Opens a new window"
		}
		return DockMenuItem{name: item.Name, description: description, action: item.Action}, nil
	case *gpui.MenuItemAction:
		return NewDockMenuItem(*item)
	default:
		return DockMenuItem{}, errors.New("Only `MenuItem::Action` is supported for dock menu on Windows.")
	}
}

type dockMenuEntry struct {
	name        gpui.SharedString
	description gpui.SharedString
}

func updateJumpList(recentWorkspaces [][]string, dockMenus []dockMenuEntry) ([][]string, error) {
	list, removed, err := createDestinationList()
	if err != nil {
		return nil, err
	}
	defer list.Release()
	if err := addRecentFolders(list, recentWorkspaces, removed); err != nil {
		return nil, err
	}
	if err := addDockMenu(list, dockMenus); err != nil {
		return nil, err
	}
	if err := list.commitList(); err != nil {
		return nil, err
	}
	return removed, nil
}

type propertyKey struct {
	fmtid com.GUID
	pid   uint32
}

var pkeyTitle = propertyKey{fmtid: com.MustGUID("{F29F85E0-4FF9-1068-AB91-08002B27B3D9}"), pid: 2}

const infotipSize = 1024

var (
	clsidDestinationList            = com.MustGUID("{77F10CF0-3DB5-4966-B520-B7C54FD35ED6}")
	clsidEnumerableObjectCollection = com.MustGUID("{2D3468C1-36A7-43B6-AC24-D3F02FD9607A}")
	clsidShellLink                  = com.MustGUID("{00021401-0000-0000-C000-000000000046}")
	iidICustomDestinationList       = com.MustGUID("{6332DEBF-87B5-4670-90C0-5E57B408A49E}")
	iidIObjectArray                 = com.MustGUID("{92CA9DCD-5622-4BBA-A805-5E9F541BD8C9}")
	iidIObjectCollection            = com.MustGUID("{5632B1A4-E38A-400A-928A-D4CD63230295}")
	iidIShellLinkW                  = com.MustGUID("{000214F9-0000-0000-C000-000000000046}")
	iidIPropertyStore               = com.MustGUID("{886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99}")
)

type iCustomDestinationList struct{ com.IUnknown }

func (l *iCustomDestinationList) beginList(slots *uint32) (*iObjectArray, error) {
	var out *iObjectArray
	err := com.CallHR(l.Ptr(), com.IUnknownSlots+1, uintptr(unsafe.Pointer(slots)), uintptr(unsafe.Pointer(&iidIObjectArray)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

func (l *iCustomDestinationList) appendCategory(category string, array *iObjectCollection) error {
	name := wideZ(category)
	return com.CallHR(l.Ptr(), com.IUnknownSlots+2, uintptr(unsafe.Pointer(name)), uintptr(array.Ptr()))
}

func (l *iCustomDestinationList) addUserTasks(array *iObjectCollection) error {
	return com.CallHR(l.Ptr(), com.IUnknownSlots+4, uintptr(array.Ptr()))
}

func (l *iCustomDestinationList) commitList() error {
	return com.CallHR(l.Ptr(), com.IUnknownSlots+5)
}

type iObjectArray struct{ com.IUnknown }

func (a *iObjectArray) getCount() (uint32, error) {
	var count uint32
	err := com.CallHR(a.Ptr(), com.IUnknownSlots+0, uintptr(unsafe.Pointer(&count)))
	return count, err
}

func (a *iObjectArray) getShellLinkAt(index uint32) (*iShellLinkW, error) {
	var out *iShellLinkW
	err := com.CallHR(a.Ptr(), com.IUnknownSlots+1, uintptr(index), uintptr(unsafe.Pointer(&iidIShellLinkW)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

type iObjectCollection struct{ iObjectArray }

func (c *iObjectCollection) addObject(object *com.IUnknown) error {
	return com.CallHR(c.Ptr(), com.IUnknownSlots+2, uintptr(unsafe.Pointer(object)))
}

type iShellLinkW struct{ com.IUnknown }

func (l *iShellLinkW) getDescription(buffer []uint16) error {
	return com.CallHR(l.Ptr(), com.IUnknownSlots+3, uintptr(unsafe.Pointer(&buffer[0])), uintptr(int32(len(buffer))))
}

func (l *iShellLinkW) setString(slot int, value string) error {
	p := wideZ(value)
	return com.CallHR(l.Ptr(), com.IUnknownSlots+slot, uintptr(unsafe.Pointer(p)))
}

func (l *iShellLinkW) setDescription(value string) error { return l.setString(4, value) }
func (l *iShellLinkW) setArguments(value string) error   { return l.setString(8, value) }
func (l *iShellLinkW) setPath(value string) error        { return l.setString(17, value) }

func (l *iShellLinkW) setIconLocation(path string, index int32) error {
	p := wideZ(path)
	return com.CallHR(l.Ptr(), com.IUnknownSlots+14, uintptr(unsafe.Pointer(p)), uintptr(index))
}

type iPropertyStore struct{ com.IUnknown }

func (s *iPropertyStore) setValue(key *propertyKey, value *com.VARIANT) error {
	return com.CallHR(s.Ptr(), com.IUnknownSlots+3, uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(value)))
}

func (s *iPropertyStore) commit() error {
	return com.CallHR(s.Ptr(), com.IUnknownSlots+4)
}

func createDestinationList() (*iCustomDestinationList, [][]string, error) {
	list, err := com.CreateInstance[iCustomDestinationList](&clsidDestinationList, com.CLSCTX_INPROC_SERVER, &iidICustomDestinationList)
	if err != nil {
		return nil, nil, err
	}

	var slots uint32
	userRemoved, err := list.beginList(&slots)
	if err != nil {
		list.Release()
		return nil, nil, err
	}
	defer userRemoved.Release()

	count, err := userRemoved.getCount()
	if err != nil {
		list.Release()
		return nil, nil, err
	}
	if count == 0 {
		return list, nil, nil
	}

	removed := make([][]string, 0, count)
	for i := range count {
		shellLink, err := userRemoved.getShellLinkAt(i)
		if err != nil {
			list.Release()
			return nil, nil, err
		}
		buffer := make([]uint16, infotipSize)
		err = shellLink.getDescription(buffer)
		shellLink.Release()
		if err != nil {
			list.Release()
			return nil, nil, err
		}
		length := slices.Index(buffer, 0)
		if length < 0 {
			length = len(buffer)
		}
		description := string(utf16.Decode(buffer[:length]))
		removed = append(removed, strings.Split(description, "\n"))
	}

	return list, removed, nil
}

func addDockMenu(list *iCustomDestinationList, dockMenus []dockMenuEntry) error {
	tasks, err := com.CreateInstance[iObjectCollection](&clsidEnumerableObjectCollection, com.CLSCTX_INPROC_SERVER, &iidIObjectCollection)
	if err != nil {
		return err
	}
	defer tasks.Release()
	for idx, menu := range dockMenus {
		argument := fmt.Sprintf("--dock-action %d", idx)
		task, err := createShellLink(argument, string(menu.description), gpui.None[string](), string(menu.name))
		if err != nil {
			return err
		}
		err = tasks.addObject(&task.IUnknown)
		task.Release()
		if err != nil {
			return err
		}
	}
	return list.addUserTasks(tasks)
}

func addRecentFolders(list *iCustomDestinationList, entries [][]string, removed [][]string) error {
	tasks, err := com.CreateInstance[iObjectCollection](&clsidEnumerableObjectCollection, com.CLSCTX_INPROC_SERVER, &iidIObjectCollection)
	if err != nil {
		return err
	}
	defer tasks.Release()

	for _, folderPath := range entries {
		if slices.ContainsFunc(removed, func(r []string) bool { return slices.Equal(r, folderPath) }) {
			continue
		}
		quoted := make([]string, len(folderPath))
		for i, path := range folderPath {
			quoted[i] = "\"" + path + "\""
		}
		argument := strings.Join(quoted, " ")

		description := strings.Join(folderPath, "\n")
		icon := gpui.Some("explorer.exe")

		names := make([]string, len(folderPath))
		for i, p := range folderPath {
			if name, ok := pathFileName(p); ok {
				names[i] = name
			} else {
				names[i] = p
			}
		}
		display := strings.Join(names, ", ")

		link, err := createShellLink(argument, description, icon, display)
		if err != nil {
			return err
		}
		err = tasks.addObject(&link.IUnknown)
		link.Release()
		if err != nil {
			return err
		}
	}

	if count, err := tasks.getCount(); err == nil && count > 0 {
		if err := list.appendCategory("Recent Folders", tasks); err != nil {
			return err
		}
	}
	return nil
}

func pathFileName(path string) (string, bool) {
	trimmed := strings.TrimRight(path, `\/`)
	for strings.HasSuffix(trimmed, `\.`) || strings.HasSuffix(trimmed, "/.") {
		trimmed = strings.TrimRight(trimmed[:len(trimmed)-1], `\/`)
	}
	if trimmed == "" || trimmed == "." {
		return "", false
	}
	if trimmed == filepath.VolumeName(trimmed) {
		return "", false
	}
	base := filepath.Base(trimmed)
	if base == ".." || base == `\` || base == "/" || base == "." {
		return "", false
	}
	return base, true
}

func createShellLink(argument, description string, icon gpui.Option[string], display string) (*iShellLinkW, error) {
	link, err := com.CreateInstance[iShellLinkW](&clsidShellLink, com.CLSCTX_INPROC_SERVER, &iidIShellLinkW)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*iShellLinkW, error) {
		link.Release()
		return nil, err
	}
	exePath, err := os.Executable()
	if err != nil {
		return fail(err)
	}
	if err := link.setPath(exePath); err != nil {
		return fail(err)
	}
	if err := link.setArguments(argument); err != nil {
		return fail(err)
	}
	if err := link.setDescription(description); err != nil {
		return fail(err)
	}
	if icon, ok := icon.Get(); ok {
		if err := link.setIconLocation(icon, 0); err != nil {
			return fail(err)
		}
	}
	store, err := com.Cast[iPropertyStore](&link.IUnknown, &iidIPropertyStore)
	if err != nil {
		return fail(err)
	}
	defer store.Release()
	title, err := com.VariantBSTR(display)
	if err != nil {
		return fail(err)
	}
	defer com.VariantClear(&title)
	if err := store.setValue(&pkeyTitle, &title); err != nil {
		return fail(err)
	}
	if err := store.commit(); err != nil {
		return fail(err)
	}
	return link, nil
}

func wideZ(s string) *uint16 {
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	wide := append(utf16.Encode([]rune(s)), 0)
	return &wide[0]
}
