# M1-linux-dbus debt

- Secret Service: oo7's sandboxed path (org.freedesktop.portal.Secret + file keyring) is not ported; credentials always go through org.freedesktop.secrets. Flatpak builds without host keyring access will fail where oo7 would use the file backend.
- Secret Service: no 30s method timeout (oo7 sets one on its own connection); calls use the caller's conn defaults.
- XDG portal: ColorScheme values other than 1/2 map to Light (ashpd may reject unknown values and skip the event).
