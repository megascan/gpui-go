# M4-apple-text handoff

- internal/apple/ct/cfx.go binds CoreFoundation API that belongs in internal/apple/cf (owned by M4-apple-ffi): CFNumberCreate/GetValue, CFArrayCreateMutable/AppendValue, CFAttributedStringCreateMutable/ReplaceString/GetLength/SetAttribute, CFLocaleCopyPreferredLanguages, CFPreferencesCopyAppValue, CFURLCopyFileSystemPath. An exit stage may move them into cf.
