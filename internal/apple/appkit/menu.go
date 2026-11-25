//go:build darwin

package appkit

import "github.com/megascan/gpui-go/internal/apple/objc"

// NSMenu wraps an NSMenu object.
type NSMenu objc.ID

// ID returns the underlying object.
func (m NSMenu) ID() objc.ID { return objc.ID(m) }

// NewNSMenu returns a +1 menu titled title.
func NewNSMenu(title string) NSMenu {
	s := objc.NSStringFromGo(title)
	defer objc.Release(s)
	return NSMenu(send[objc.ID](objc.Alloc(class("NSMenu")), "initWithTitle:", s))
}

// AddItem appends an item.
func (m NSMenu) AddItem(item NSMenuItem) { sendVoid(m.ID(), "addItem:", item.ID()) }

// SetDelegate sets the menu delegate.
func (m NSMenu) SetDelegate(delegate objc.ID) { sendVoid(m.ID(), "setDelegate:", delegate) }

// SetAutoenablesItems sets whether items are validated automatically.
func (m NSMenu) SetAutoenablesItems(flag bool) { sendVoid(m.ID(), "setAutoenablesItems:", flag) }

// RemoveAllItems removes every item.
func (m NSMenu) RemoveAllItems() { sendVoid(m.ID(), "removeAllItems") }

// NumberOfItems returns the number of items.
func (m NSMenu) NumberOfItems() int { return send[int](m.ID(), "numberOfItems") }

// ItemAtIndex returns the item at index.
func (m NSMenu) ItemAtIndex(index int) NSMenuItem {
	return NSMenuItem(send[objc.ID](m.ID(), "itemAtIndex:", index))
}

// NSMenuItemModifierMask is the key equivalent modifier mask of a menu item.
type NSMenuItemModifierMask = NSEventModifierFlags

// NSMenuItem wraps an NSMenuItem object.
type NSMenuItem objc.ID

// ID returns the underlying object.
func (i NSMenuItem) ID() objc.ID { return objc.ID(i) }

// NewNSMenuItem returns a +1 menu item.
func NewNSMenuItem(title string, action objc.SEL, keyEquivalent string) NSMenuItem {
	t := objc.NSStringFromGo(title)
	defer objc.Release(t)
	k := objc.NSStringFromGo(keyEquivalent)
	defer objc.Release(k)
	return NSMenuItem(send[objc.ID](objc.Alloc(class("NSMenuItem")), "initWithTitle:action:keyEquivalent:", t, action, k))
}

// NSMenuItemSeparator returns a separator item.
func NSMenuItemSeparator() NSMenuItem {
	return NSMenuItem(sendClass[objc.ID]("NSMenuItem", "separatorItem"))
}

// SetSubmenu sets the submenu.
func (i NSMenuItem) SetSubmenu(menu NSMenu) { sendVoid(i.ID(), "setSubmenu:", menu.ID()) }

// SetTarget sets the action target.
func (i NSMenuItem) SetTarget(target objc.ID) { sendVoid(i.ID(), "setTarget:", target) }

// SetTag sets the tag.
func (i NSMenuItem) SetTag(tag int) { sendVoid(i.ID(), "setTag:", tag) }

// Tag returns the tag.
func (i NSMenuItem) Tag() int { return send[int](i.ID(), "tag") }

// SetEnabled sets whether the item is enabled.
func (i NSMenuItem) SetEnabled(flag bool) { sendVoid(i.ID(), "setEnabled:", flag) }

// SetState sets the check state.
func (i NSMenuItem) SetState(state int) { sendVoid(i.ID(), "setState:", state) }

// SetKeyEquivalentModifierMask sets the key equivalent modifiers.
func (i NSMenuItem) SetKeyEquivalentModifierMask(mask NSMenuItemModifierMask) {
	sendVoid(i.ID(), "setKeyEquivalentModifierMask:", mask)
}

// SetAllowsAutomaticKeyEquivalentLocalization sets whether AppKit relocalises the key equivalent.
func (i NSMenuItem) SetAllowsAutomaticKeyEquivalentLocalization(flag bool) {
	sendVoid(i.ID(), "setAllowsAutomaticKeyEquivalentLocalization:", flag)
}

// NSControlStateValueOn is the on state of a control or menu item.
const NSControlStateValueOn = 1
