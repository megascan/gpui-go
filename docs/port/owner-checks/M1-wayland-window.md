# M1-wayland-window owner checks

- YOU: on a KDE Plasma (< 6.7, kde-blur) and a compositor with ext-background-effect, open an example with `WindowBackgroundAppearanceBlurred` and check the blur follows the rounded client-decoration corners.
- YOU: on a layer-shell compositor (Hyprland / Sway), open a layer-shell window and an anchored popup with grab, check placement, dismissal on outside click and `xdg_popup.reposition` on resize.
