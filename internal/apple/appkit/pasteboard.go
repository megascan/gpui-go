//go:build darwin

package appkit

import "github.com/megascan/gpui-go/internal/apple/objc"

// NSPasteboard wraps an NSPasteboard object.
type NSPasteboard objc.ID

// ID returns the underlying object.
func (p NSPasteboard) ID() objc.ID { return objc.ID(p) }

// NSPasteboardGeneral returns the general pasteboard.
func NSPasteboardGeneral() NSPasteboard {
	return NSPasteboard(sendClass[objc.ID]("NSPasteboard", "generalPasteboard"))
}

// NSPasteboardWithName returns the pasteboard named name.
func NSPasteboardWithName(name objc.ID) NSPasteboard {
	return NSPasteboard(sendClass[objc.ID]("NSPasteboard", "pasteboardWithName:", name))
}

// NSPasteboardWithUniqueName returns a new pasteboard with a unique name.
func NSPasteboardWithUniqueName() NSPasteboard {
	return NSPasteboard(sendClass[objc.ID]("NSPasteboard", "pasteboardWithUniqueName"))
}

// NSPasteboardTypeString returns NSPasteboardTypeString.
func NSPasteboardTypeString() objc.ID { return Constant("NSPasteboardTypeString") }

// NSPasteboardTypePNG returns NSPasteboardTypePNG.
func NSPasteboardTypePNG() objc.ID { return Constant("NSPasteboardTypePNG") }

// NSPasteboardTypeTIFF returns NSPasteboardTypeTIFF.
func NSPasteboardTypeTIFF() objc.ID { return Constant("NSPasteboardTypeTIFF") }

// NSPasteboardTypeFileURL returns NSPasteboardTypeFileURL.
func NSPasteboardTypeFileURL() objc.ID { return Constant("NSPasteboardTypeFileURL") }

// NSPasteboardNameFind returns NSPasteboardNameFind.
func NSPasteboardNameFind() objc.ID { return Constant("NSPasteboardNameFind") }

// ChangeCount returns the pasteboard's change count.
func (p NSPasteboard) ChangeCount() int { return send[int](p.ID(), "changeCount") }

// ClearContents clears the pasteboard and returns the new change count.
func (p NSPasteboard) ClearContents() int { return send[int](p.ID(), "clearContents") }

// Types returns the pasteboard's types.
func (p NSPasteboard) Types() []objc.ID { return objc.NSArrayToIDs(send[objc.ID](p.ID(), "types")) }

// PasteboardItems returns the pasteboard items.
func (p NSPasteboard) PasteboardItems() []objc.ID {
	return objc.NSArrayToIDs(send[objc.ID](p.ID(), "pasteboardItems"))
}

// DataForType returns the NSData for a type or nil.
func (p NSPasteboard) DataForType(dataType objc.ID) objc.ID {
	return send[objc.ID](p.ID(), "dataForType:", dataType)
}

// StringForType returns the NSString for a type or nil.
func (p NSPasteboard) StringForType(dataType objc.ID) objc.ID {
	return send[objc.ID](p.ID(), "stringForType:", dataType)
}

// SetDataForType stores NSData for a type.
func (p NSPasteboard) SetDataForType(data, dataType objc.ID) bool {
	return send[bool](p.ID(), "setData:forType:", data, dataType)
}

// SetStringForType stores an NSString for a type.
func (p NSPasteboard) SetStringForType(str, dataType objc.ID) bool {
	return send[bool](p.ID(), "setString:forType:", str, dataType)
}

// DeclareTypes declares the pasteboard's types.
func (p NSPasteboard) DeclareTypes(types []objc.ID, owner objc.ID) int {
	array := objc.NSArrayFromIDs(types)
	defer objc.Release(array)
	return send[int](p.ID(), "declareTypes:owner:", array, owner)
}

// WriteObjects writes NSPasteboardWriting objects.
func (p NSPasteboard) WriteObjects(objects []objc.ID) bool {
	array := objc.NSArrayFromIDs(objects)
	defer objc.Release(array)
	return send[bool](p.ID(), "writeObjects:", array)
}

// ReleaseGlobally releases the pasteboard's resources.
func (p NSPasteboard) ReleaseGlobally() { sendVoid(p.ID(), "releaseGlobally") }
