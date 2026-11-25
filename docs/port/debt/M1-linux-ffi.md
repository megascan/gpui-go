# M1-linux-ffi debt

- internal/linux/xkb: Keymap.key_for_each and context include-path / logging setters of xkbcommon-rs are not bound (unused by gpui_linux). Add if a later stage needs them.
- internal/linux/xkb/keysyms.go holds Latin-1, 0xfe00-0xffff and the eight XF86 keysyms gpui_linux uses (C names, e.g. KEY_XF86Back for xkeysym's XF86_Back), not the full header.
