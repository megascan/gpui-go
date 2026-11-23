# Parity matrix

Upstream: zed-industries/zed @ `6501df15765d66d6c2a2b66b1eb22434ccef3f86`, read from `_ref/` (all upstream paths below are relative to it). Design and rationale: [DESIGN.md](DESIGN.md).

## 1. What this is

The matrix maps every upstream public item and every upstream test to a Go symbol, a Go test, or a registered deviation. It is split so that parallel stages never edit the same file:

- This file: totals, the naming rules, the file-level map (every upstream `.rs` file to its Go file), the deviation registry, examples, external ports.
- `parity/<area>.md`: one row per public item and per test, grouped by upstream file. One area file per ownership unit.
- `parity/styled-methods.tsv`: the 3099 `Styled` methods the style macros generate, one per line with the Go name.
- `parity/_collisions.md`: Go names that more than one upstream item maps to.

All rows start as `todo`. Nothing in this repository is ported yet; the Go column is the name the rules in section 3 produce, not a symbol that exists.

## 2. Totals

| Crate | Files | LOC | Public items | Tests |
|---|---|---|---|---|
| gpui | 154 | 109267 | 6071 | 578 |
| gpui_platform | 5 | 2820 | 10 | 30 |
| gpui_linux | 28 | 17620 | 91 | 55 |
| gpui_wgpu | 6 | 6335 | 57 | 48 |
| gpui_macos | 12 | 12007 | 60 | 18 |
| gpui_apple | 10 | 4750 | 53 | 21 |
| gpui_windows | 21 | 13443 | 6 | 20 |
| gpui_web | 21 | 8064 | 23 | 21 |
| gpui_ios | 10 | 3705 | 11 | 12 |
| gpui_macros | 15 | 3631 | 19 | 3 |
| gpui_util | 2 | 846 | 37 | 2 |
| gpui_shared_string | 1 | 223 | 4 | 0 |
| gpui_tokio | 1 | 100 | 6 | 0 |
| refineable | 2 | 680 | 15 | 0 |
| scheduler | 5 | 3215 | 100 | 31 |
| sum_tree | 4 | 3555 | 116 | 11 |
| collections | 3 | 418 | 27 | 19 |
| Total | 300 | 190679 | 6706 | 869 |

Public items in gpui include the 3099 macro-generated `Styled` methods. Test attributes: 630 `#[test]`, 178 `#[gpui::test]`, 30 `#[gpui::bench]`, 27 `#[crate::test]`, 3 `#[perf]`, 1 `#[gpui::property_test]`; 5 carry `#[ignore]`.

| Area file | Upstream scope | Milestone | Go paths owned | Files | Public items | Tests |
|---|---|---|---|---|---|---|
| [parity/foundation.md](parity/foundation.md) | collections, gpui_util, gpui_shared_string, gpui_tokio, refineable | M0 | `collections/`, `gpuiutil/`, `refineable/`, `internal/cmd/refinegen`, root `shared_string.go` | 9 | 89 | 21 |
| [parity/sumtree.md](parity/sumtree.md) | sum_tree | M0 | `sumtree/` | 4 | 116 | 11 |
| [parity/scheduler.md](parity/scheduler.md) | scheduler | M0 | `scheduler/` | 5 | 100 | 31 |
| [parity/macros.md](parity/macros.md) | gpui_macros | M0 | `internal/cmd/{stylegen,actiongen}`, root helpers | 15 | 19 | 3 |
| [parity/core-geometry.md](parity/core-geometry.md) | gpui: geometry, color, bounds tree, util, arena, shared_uri, crate root | M0 | root | 9 | 241 | 26 |
| [parity/core-app.md](parity/core-app.md) | gpui: app, entities, contexts, executor, subscriptions, actions, globals, assets, test harness | M0 | root | 19 | 614 | 48 |
| [parity/core-input.md](parity/core-input.md) | gpui: keymap, key dispatch, interactive events, input handler, gestures, tab stops | M0 | root | 10 | 185 | 111 |
| [parity/core-platform.md](parity/core-platform.md) | gpui: platform interfaces, TestPlatform, dispatchers | M0 | root | 13 | 515 | 32 |
| [parity/core-window.md](parity/core-window.md) | gpui: window, element, view, inspector, a11y, native views, prompts | M0 | root | 9 | 373 | 74 |
| [parity/core-style.md](parity/core-style.md) | gpui: style, styled, taffy bridge, scene, paths, spring | M0 | root, `internal/cmd/stylegen` | 7 | 3365 | 24 |
| [parity/core-text.md](parity/core-text.md) | gpui: text system, line layout and wrapping, svg renderer | M0, M2 | root | 7 | 144 | 58 |
| [parity/core-elements.md](parity/core-elements.md) | gpui: elements/* | M0, M3 | root | 14 | 393 | 102 |
| [parity/core-profiler.md](parity/core-profiler.md) | gpui: profiler, debug overlay, hot reload, randomized element tree, bench context | M3 | root | 10 | 241 | 82 |
| [parity/examples-m2.md](parity/examples-m2.md) | gpui: examples (M1, M2 subset) | M1, M2 | `examples/` (hello_world, text, svg, image, internal/support) | 5 | 0 | 0 |
| [parity/examples-a.md](parity/examples-a.md) | gpui: examples | M3 | `examples/` (active_state_bug to popover) | 13 | 0 | 0 |
| [parity/examples-b.md](parity/examples-b.md) | gpui: examples | M3 | `examples/` (text_layout to painting) | 11 | 0 | 0 |
| [parity/examples-c.md](parity/examples-c.md) | gpui: examples | M3 | `examples/` (window to native_view_appkit_field) | 15 | 0 | 0 |
| [parity/examples-d.md](parity/examples-d.md) | gpui: examples (view_example, testing, input, animation) | M3 | `examples/view_example/`, `examples/testing/`, `examples/input/`, `examples/animation/` | 8 | 0 | 15 |
| [parity/core-tests.md](parity/core-tests.md) | gpui: integration tests and benches | M0, M3 | root `_test` files, `bench_focus_cleanup_test.go` | 4 | 0 | 6 |
| [parity/platform-select.md](parity/platform-select.md) | gpui_platform | M1 | `platform/` | 5 | 10 | 30 |
| [parity/linux-common.md](parity/linux-common.md) | gpui_linux: shared state (dispatcher, platform, headless, keyboard, text system) | M1 | `platform/linux/internal/common/`, `platform/linux/platform.go` | 6 | 5 | 17 |
| [parity/linux-dbus.md](parity/linux-dbus.md) | gpui_linux: D-Bus services (power, notifications, portal) | M1 | `platform/linux/internal/dbus/` | 3 | 3 | 4 |
| [parity/linux-platform.md](parity/linux-platform.md) | gpui_linux: entry points, display connection | M1 | `platform/linux/linux.go`, `platform/linux/display_connection.go` | 3 | 2 | 0 |
| [parity/linux-wayland.md](parity/linux-wayland.md) | gpui_linux: Wayland back end | M1 | `platform/linux/wayland/` | 9 | 65 | 22 |
| [parity/linux-x11.md](parity/linux-x11.md) | gpui_linux: X11 back end | M1 | `platform/linux/x11/` | 7 | 16 | 12 |
| [parity/wgpu.md](parity/wgpu.md) | gpui_wgpu: renderer, atlas, context | M1, M4 | `render/wgpu` | 4 | 52 | 18 |
| [parity/gotext.md](parity/gotext.md) | gpui_wgpu: cosmic-text system | M2 | `text/gotext` | 2 | 5 | 30 |
| [parity/apple.md](parity/apple.md) | gpui_apple | M4 | `platform/apple` | 10 | 53 | 21 |
| [parity/macos.md](parity/macos.md) | gpui_macos | M4 | `platform/macos` | 12 | 60 | 18 |
| [parity/ios.md](parity/ios.md) | gpui_ios | M4 | `platform/ios` | 10 | 11 | 12 |
| [parity/windows.md](parity/windows.md) | gpui_windows | M4 | `platform/windows` | 21 | 6 | 20 |
| [parity/web.md](parity/web.md) | gpui_web | M4 | `platform/web` | 21 | 23 | 21 |

## 3. Naming and file rules

The rules are deterministic so that two builders derive the same Go name for the same upstream item without seeing each other's work. DESIGN.md section 5.1 gives the reasons.

Symbols:

| # | Upstream | Go |
|---|---|---|
| N1 | type name | same, with whole-word initialisms upper-cased: ID, URL, URI, HTTP, JSON, UTF8, UTF16, IME, GPU, CPU, UUID (`EntityId` -> `EntityID`) |
| N2 | free `fn foo_bar` | `FooBar` (same initialisms per snake segment) |
| N3 | free fn whose Go name equals a type in the same Go package | `New<Type>`; `Type::new` maps to the same function |
| N4 | `rems`, `radians`, `percentage` | the Go conversion `Rems(x)` etc. |
| N5 | method taking `self` | `(Type).FooBar` |
| N6 | associated fn without `self` | `new` -> `NewType`; `new_x` -> `NewTypeX`; `default` -> `DefaultType`; otherwise `TypeFooBar` |
| N7 | inherent impl on one concrete instantiation, e.g. `impl Edges<Length>` | numeric instantiations (`Pixels`, `ScaledPixels`, `DevicePixels`, `Rems`, primitive numbers): method on the generic numeric type. Others: free function `<Type><Inst><Name>` |
| N8 | trait | interface of the same name; the five fluent traits become mixins: `Styled[S]`, `Interactive[S]`, `StatefulInteractive[S]`, `Parent[S]`, `Fluent[S]` |
| N9 | trait method | `Trait.FooBar`, or `(Mixin[S]).FooBar` |
| N10 | `SCREAMING_CONST`, statics | `ScreamingConst` |
| N11 | snake boundary between two digits (`w_1_2`) | underscore kept: `W1_2`; `w_12` is `W12` |
| N12 | enum variants (not listed as rows) | `<Enum><Variant>` constants, or one struct per variant for data-carrying enums |
| N13 | `#[cfg]` alternatives of one item | one Go symbol, bodies in per-GOOS files |
| N14 | test `fn test_foo` or `fn foo` in `bar.rs` | `TestBar_Foo`; benches `BenchmarkBar_Foo`; a named inner test module is inserted after the file stem |

Files:

| # | Rule |
|---|---|
| F1 | One upstream file maps to one Go file. gpui: the path under `src/` with `/` replaced by `_` (`app/entity_map.rs` -> `app_entity_map.go`). Other crates: the path under `src/` inside the crate's Go package. |
| F2 | A path segment `test` or `tests` becomes `testing` (`platform/test/window.rs` -> `platform_testing_window.go`), and a stem ending in `_test` gets `ing` appended, so no production file ends in `_test.go`. |
| F3 | A stem that would end in a GOOS or GOARCH word gets `_impl` appended, so no file picks up an implicit build constraint. |
| F4 | Tests for `x.go` live in `x_test.go` in the same package. Upstream `tests/` directories and `benches/` become `integration_*_test.go` and `bench_*_test.go` in an external test package. |
| F5 | Generated files end in `_gen.go`. |
| F6 | No Go file name starts with `_` or `.` (the go tool ignores them). The doc-only modules `gpui/src/_accessibility.rs` and `gpui/src/_ownership_and_data_flow.rs` (`cfg(doc)`, `gpui/src/gpui.rs:96-99`) become pages under lowercase `docs/` (never `DOCS/`: the macOS and Windows runners have case-insensitive file systems). |

## 4. File map

Every upstream `.rs` file, its Go target, and how many rows it has in its area file. `-` means no Go file (build scripts, crates not ported).

### foundation

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_util/src/arc_cow.rs` | 141 | `-` | 1 | 0 |
| `gpui_util/src/lib.rs` | 705 | `gpuiutil/lib.go` | 36 | 2 |
| `gpui_shared_string/gpui_shared_string.rs` | 223 | `shared_string.go` | 4 | 0 |
| `gpui_tokio/src/gpui_tokio.rs` | 100 | `-` | 6 | 0 |
| `refineable/derive_refineable/src/derive_refineable.rs` | 548 | `internal/cmd/refinegen/main.go` | 1 | 0 |
| `refineable/src/refineable.rs` | 132 | `refineable/refineable.go` | 14 | 0 |
| `collections/src/collections.rs` | 15 | `collections/collections.go` | 6 | 0 |
| `collections/src/vecmap.rs` | 192 | `collections/vecmap.go` | 21 | 0 |
| `collections/src/vecmap_tests.rs` | 211 | `collections/vecmap_test.go` | 0 | 19 |

### sumtree

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `sum_tree/src/cursor.rs` | 861 | `sumtree/cursor.go` | 28 | 0 |
| `sum_tree/src/property_test.rs` | 32 | `sumtree/property_testing.go` | 1 | 0 |
| `sum_tree/src/sum_tree.rs` | 2131 | `sumtree/sum_tree.go` | 53 | 6 |
| `sum_tree/src/tree_map.rs` | 531 | `sumtree/tree_map.go` | 34 | 5 |

### scheduler

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `scheduler/src/clock.rs` | 49 | `scheduler/clock.go` | 7 | 0 |
| `scheduler/src/executor.rs` | 859 | `scheduler/executor.go` | 38 | 0 |
| `scheduler/src/scheduler.rs` | 266 | `scheduler/scheduler.go` | 21 | 0 |
| `scheduler/src/test_scheduler.rs` | 971 | `scheduler/test_scheduler.go` | 34 | 0 |
| `scheduler/src/tests.rs` | 1070 | `scheduler/scheduler_test.go` | 0 | 31 |

### macros

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_macros/src/bench.rs` | 304 | `bench.go` | 0 | 0 |
| `gpui_macros/src/derive_action.rs` | 211 | `internal/cmd/actiongen/main.go` | 0 | 0 |
| `gpui_macros/src/derive_app_context.rs` | 119 | `gpui.go` | 0 | 0 |
| `gpui_macros/src/derive_inspector_reflection.rs` | 305 | `inspector_reflection.go` | 0 | 0 |
| `gpui_macros/src/derive_into_element.rs` | 30 | `element.go` | 0 | 0 |
| `gpui_macros/src/derive_render.rs` | 21 | `element.go` | 0 | 0 |
| `gpui_macros/src/derive_visual_context.rs` | 73 | `gpui.go` | 0 | 0 |
| `gpui_macros/src/gpui_macros.rs` | 331 | `internal/cmd/(index)` | 19 | 0 |
| `gpui_macros/src/property_test.rs` | 273 | `testing.go` | 0 | 0 |
| `gpui_macros/src/register_action.rs` | 47 | `action.go` | 0 | 0 |
| `gpui_macros/src/styles.rs` | 1417 | `internal/cmd/stylegen/main.go` | 0 | 0 |
| `gpui_macros/src/test.rs` | 347 | `testing.go` | 0 | 0 |
| `gpui_macros/tests/derive_context.rs` | 13 | `macros_derive_context_test.go` | 0 | 1 |
| `gpui_macros/tests/derive_inspector_reflection.rs` | 133 | `macros_derive_inspector_reflection_test.go` | 0 | 1 |
| `gpui_macros/tests/render_test.rs` | 7 | `macros_render_test_test.go` | 0 | 1 |

### core-geometry

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/src/arena.rs` | 401 | `arena.go` | 9 | 7 |
| `gpui/src/bounds_tree.rs` | 500 | `bounds_tree.go` | 0 | 2 |
| `gpui/src/color.rs` | 1070 | `color.go` | 46 | 10 |
| `gpui/src/colors.rs` | 123 | `colors.go` | 9 | 0 |
| `gpui/src/geometry.rs` | 4165 | `geometry.go` | 139 | 3 |
| `gpui/src/gpui.rs` | 355 | `gpui.go` | 27 | 0 |
| `gpui/src/prelude.rs` | 9 | `prelude.go` | 0 | 0 |
| `gpui/src/shared_uri.rs` | 25 | `shared_uri.go` | 1 | 0 |
| `gpui/src/util.rs` | 252 | `util.go` | 10 | 4 |

### core-app

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/src/_ownership_and_data_flow.rs` | 140 | `-` | 0 | 0 |
| `gpui/src/action.rs` | 456 | `action.go` | 19 | 0 |
| `gpui/src/app.rs` | 4031 | `app.go` | 209 | 5 |
| `gpui/src/asset_cache.rs` | 358 | `asset_cache.go` | 5 | 5 |
| `gpui/src/assets.rs` | 133 | `assets.go` | 11 | 1 |
| `gpui/src/executor.rs` | 626 | `executor.go` | 43 | 2 |
| `gpui/src/global.rs` | 75 | `global.go` | 6 | 0 |
| `gpui/src/platform_scheduler.rs` | 457 | `platform_scheduler.go` | 3 | 6 |
| `gpui/src/queue.rs` | 541 | `queue.go` | 15 | 3 |
| `gpui/src/seeds.rs` | 67 | `seeds.go` | 1 | 1 |
| `gpui/src/subscription.rs` | 559 | `subscription.go` | 4 | 8 |
| `gpui/src/test.rs` | 172 | `testing.go` | 6 | 0 |
| `gpui/src/app/async_context.rs` | 559 | `app_async_context.go` | 24 | 0 |
| `gpui/src/app/context.rs` | 938 | `app_context.go` | 44 | 0 |
| `gpui/src/app/entity_map.rs` | 1421 | `app_entity_map.go` | 31 | 5 |
| `gpui/src/app/headless_app_context.rs` | 293 | `app_headless_app_context.go` | 16 | 0 |
| `gpui/src/app/test_app.rs` | 647 | `app_test_app.go` | 50 | 4 |
| `gpui/src/app/test_context.rs` | 1562 | `app_test_context.go` | 94 | 8 |
| `gpui/src/app/visual_test_context.rs` | 485 | `app_visual_test_context.go` | 33 | 0 |

### core-input

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/src/gestures.rs` | 2331 | `gestures.go` | 17 | 37 |
| `gpui/src/input.rs` | 611 | `input.go` | 18 | 2 |
| `gpui/src/interactive.rs` | 998 | `interactive.go` | 56 | 2 |
| `gpui/src/key_dispatch.rs` | 2200 | `key_dispatch.go` | 0 | 25 |
| `gpui/src/keymap.rs` | 1143 | `keymap.go` | 12 | 24 |
| `gpui/src/tab_stop.rs` | 501 | `tab_stop.go` | 1 | 10 |
| `gpui/src/keymap/binding.rs` | 166 | `keymap_binding.go` | 12 | 0 |
| `gpui/src/keymap/context.rs` | 888 | `keymap_context.go` | 19 | 11 |
| `gpui/src/platform/keyboard.rs` | 41 | `platform_keyboard.go` | 7 | 0 |
| `gpui/src/platform/keystroke.rs` | 813 | `platform_keystroke.go` | 43 | 0 |

### core-platform

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/src/platform.rs` | 4116 | `platform.go` | 426 | 22 |
| `gpui/src/platform/app_menu.rs` | 426 | `platform_app_menu.go` | 23 | 2 |
| `gpui/src/platform/layer_shell.rs` | 83 | `platform_layer_shell.go` | 4 | 0 |
| `gpui/src/platform/platform_info.rs` | 49 | `platform_platform_info.go` | 4 | 0 |
| `gpui/src/platform/popup.rs` | 134 | `platform_popup.go` | 4 | 0 |
| `gpui/src/platform/scap_screen_capture.rs` | 325 | `platform_scap_screen_capture.go` | 1 | 0 |
| `gpui/src/platform/test.rs` | 12 | `platform_testing.go` | 0 | 0 |
| `gpui/src/platform/threaded_dispatcher.rs` | 744 | `platform_threaded_dispatcher.go` | 6 | 8 |
| `gpui/src/platform/visual_test.rs` | 280 | `platform_visual_testing.go` | 3 | 0 |
| `gpui/src/platform/test/dispatcher.rs` | 148 | `platform_testing_dispatcher.go` | 15 | 0 |
| `gpui/src/platform/test/display.rs` | 44 | `platform_testing_display.go` | 0 | 0 |
| `gpui/src/platform/test/platform.rs` | 861 | `platform_testing_platform.go` | 3 | 0 |
| `gpui/src/platform/test/window.rs` | 742 | `platform_testing_window.go` | 26 | 0 |

### core-window

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/src/_accessibility.rs` | 295 | `-` | 0 | 0 |
| `gpui/src/element.rs` | 1000 | `element.go` | 32 | 3 |
| `gpui/src/inspector.rs` | 271 | `inspector.go` | 10 | 0 |
| `gpui/src/view.rs` | 667 | `view.go` | 15 | 0 |
| `gpui/src/window.rs` | 11169 | `window.go` | 299 | 47 |
| `gpui/src/window/a11y.rs` | 905 | `window_a11y.go` | 4 | 13 |
| `gpui/src/window/native_view.rs` | 897 | `window_native_view.go` | 6 | 11 |
| `gpui/src/window/prompts.rs` | 232 | `window_prompts.go` | 7 | 0 |
| `gpui/src/window/a11y/debug.rs` | 330 | `window_a11y_debug.go` | 0 | 0 |

### core-style

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/src/path_builder.rs` | 577 | `path_builder.go` | 22 | 0 |
| `gpui/src/path_winding.rs` | 842 | `path_winding.go` | 0 | 5 |
| `gpui/src/scene.rs` | 1475 | `scene.go` | 44 | 4 |
| `gpui/src/spring.rs` | 830 | `spring.go` | 28 | 11 |
| `gpui/src/style.rs` | 1529 | `style.go` | 44 | 3 |
| `gpui/src/styled.rs` | 904 | `styled.go` | 3216 | 0 |
| `gpui/src/taffy.rs` | 787 | `taffy.go` | 11 | 1 |

### core-text

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/src/svg_renderer.rs` | 854 | `svg_renderer.go` | 11 | 10 |
| `gpui/src/text_system.rs` | 1401 | `text_system.go` | 72 | 4 |
| `gpui/src/text_system/font_fallbacks.rs` | 21 | `text_system_font_fallbacks.go` | 3 | 0 |
| `gpui/src/text_system/font_features.rs` | 163 | `text_system_font_features.go` | 4 | 0 |
| `gpui/src/text_system/line.rs` | 1871 | `text_system_line.go` | 20 | 12 |
| `gpui/src/text_system/line_layout.rs` | 1091 | `text_system_line_layout.go` | 22 | 11 |
| `gpui/src/text_system/line_wrapper.rs` | 2068 | `text_system_line_wrapper.go` | 12 | 21 |

### core-elements

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/src/elements/anchored.rs` | 571 | `elements_anchored.go` | 12 | 4 |
| `gpui/src/elements/animation.rs` | 1142 | `elements_animation.go` | 22 | 14 |
| `gpui/src/elements/canvas.rs` | 87 | `elements_canvas.go` | 2 | 0 |
| `gpui/src/elements/container_query.rs` | 118 | `elements_container_query.go` | 2 | 0 |
| `gpui/src/elements/deferred.rs` | 311 | `elements_deferred.go` | 4 | 2 |
| `gpui/src/elements/div.rs` | 5865 | `elements_div.go` | 176 | 28 |
| `gpui/src/elements/image_cache.rs` | 322 | `elements_image_cache.go` | 21 | 0 |
| `gpui/src/elements/img.rs` | 910 | `elements_img.go` | 19 | 5 |
| `gpui/src/elements/list.rs` | 3883 | `elements_list.go` | 43 | 41 |
| `gpui/src/elements/mod.rs` | 27 | `elements_mod.go` | 0 | 0 |
| `gpui/src/elements/surface.rs` | 126 | `elements_surface.go` | 4 | 0 |
| `gpui/src/elements/svg.rs` | 729 | `elements_svg.go` | 29 | 4 |
| `gpui/src/elements/text.rs` | 1312 | `elements_text.go` | 33 | 2 |
| `gpui/src/elements/uniform_list.rs` | 928 | `elements_uniform_list.go` | 26 | 2 |

### core-profiler

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/src/bench.rs` | 65 | `bench.go` | 2 | 0 |
| `gpui/src/debug_overlay.rs` | 428 | `debug_overlay.go` | 2 | 6 |
| `gpui/src/hot_reload.rs` | 248 | `hot_reload.go` | 0 | 2 |
| `gpui/src/profiler.rs` | 2226 | `profiler.go` | 88 | 15 |
| `gpui/src/randomized_element_tree.rs` | 2058 | `randomized_element_tree.go` | 50 | 6 |
| `gpui/src/app/bench_context.rs` | 1614 | `app_bench_context.go` | 32 | 6 |
| `gpui/src/profiler/actions.rs` | 213 | `profiler_actions.go` | 13 | 0 |
| `gpui/src/profiler/hang.rs` | 1865 | `profiler_hang.go` | 16 | 20 |
| `gpui/src/profiler/journal.rs` | 3544 | `profiler_journal.go` | 34 | 27 |
| `gpui/src/profiler/hang/monitor.rs` | 120 | `profiler_hang_monitor.go` | 4 | 0 |

### examples-m2

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/examples/hello_world.rs` | 127 | `examples/hello_world/main.go` | 0 | 0 |
| `gpui/examples/text.rs` | 432 | `examples/text/main.go` | 0 | 0 |
| `gpui/examples/example_support/fonts.rs` | 44 | `examples/internal/support/fonts.go` | 0 | 0 |
| `gpui/examples/image/image.rs` | 220 | `examples/image_example/main.go` | 0 | 0 |
| `gpui/examples/svg/svg.rs` | 108 | `examples/svg_example/main.go` | 0 | 0 |

### examples-a

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/examples/active_state_bug.rs` | 47 | `examples/active_state_bug/main.go` | 0 | 0 |
| `gpui/examples/anchor.rs` | 207 | `examples/anchor/main.go` | 0 | 0 |
| `gpui/examples/data_table.rs` | 494 | `examples/data_table/main.go` | 0 | 0 |
| `gpui/examples/gradient.rs` | 278 | `examples/gradient/main.go` | 0 | 0 |
| `gpui/examples/grid_layout.rs` | 95 | `examples/grid_layout/main.go` | 0 | 0 |
| `gpui/examples/list_example.rs` | 176 | `examples/list_example/main.go` | 0 | 0 |
| `gpui/examples/opacity.rs` | 195 | `examples/opacity/main.go` | 0 | 0 |
| `gpui/examples/paths_bench.rs` | 113 | `examples/paths_bench/main.go` | 0 | 0 |
| `gpui/examples/pattern.rs` | 136 | `examples/pattern/main.go` | 0 | 0 |
| `gpui/examples/popover.rs` | 197 | `examples/popover/main.go` | 0 | 0 |
| `gpui/examples/scrollable.rs` | 78 | `examples/scrollable/main.go` | 0 | 0 |
| `gpui/examples/tree.rs` | 64 | `examples/tree/main.go` | 0 | 0 |
| `gpui/examples/uniform_list.rs` | 71 | `examples/uniform_list/main.go` | 0 | 0 |

### examples-b

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/examples/a11y.rs` | 272 | `examples/a11y/main.go` | 0 | 0 |
| `gpui/examples/drag_drop.rs` | 158 | `examples/drag_drop/main.go` | 0 | 0 |
| `gpui/examples/focus_visible.rs` | 235 | `examples/focus_visible/main.go` | 0 | 0 |
| `gpui/examples/mouse_pressure.rs` | 87 | `examples/mouse_pressure/main.go` | 0 | 0 |
| `gpui/examples/ownership_post.rs` | 56 | `examples/ownership_post/main.go` | 0 | 0 |
| `gpui/examples/painting.rs` | 533 | `examples/painting/main.go` | 0 | 0 |
| `gpui/examples/shadow.rs` | 622 | `examples/shadow/main.go` | 0 | 0 |
| `gpui/examples/svg_animation.rs` | 222 | `examples/svg_animation/main.go` | 0 | 0 |
| `gpui/examples/tab_stop.rs` | 220 | `examples/tab_stop/main.go` | 0 | 0 |
| `gpui/examples/text_layout.rs` | 117 | `examples/text_layout/main.go` | 0 | 0 |
| `gpui/examples/text_wrapper.rs` | 144 | `examples/text_wrapper/main.go` | 0 | 0 |

### examples-c

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/examples/gif_viewer.rs` | 62 | `examples/gif_viewer/main.go` | 0 | 0 |
| `gpui/examples/image_gallery.rs` | 300 | `examples/image_gallery/main.go` | 0 | 0 |
| `gpui/examples/image_loading.rs` | 232 | `examples/image_loading/main.go` | 0 | 0 |
| `gpui/examples/layer_shell.rs` | 101 | `examples/layer_shell/main.go` | 0 | 0 |
| `gpui/examples/move_entity_between_windows.rs` | 160 | `examples/move_entity_between_windows/main.go` | 0 | 0 |
| `gpui/examples/native_view.rs` | 571 | `examples/native_view/main.go` | 0 | 0 |
| `gpui/examples/native_view_appkit_field.rs` | 226 | `examples/native_view_appkit_field/main.go` | 0 | 0 |
| `gpui/examples/on_window_close_quit.rs` | 103 | `examples/on_window_close_quit/main.go` | 0 | 0 |
| `gpui/examples/set_menus.rs` | 134 | `examples/set_menus/main.go` | 0 | 0 |
| `gpui/examples/system_notifications.rs` | 155 | `examples/system_notifications/main.go` | 0 | 0 |
| `gpui/examples/toggle_headless.rs` | 309 | `examples/toggle_headless/main.go` | 0 | 0 |
| `gpui/examples/window.rs` | 355 | `examples/window/main.go` | 0 | 0 |
| `gpui/examples/window_movable.rs` | 131 | `examples/window_movable/main.go` | 0 | 0 |
| `gpui/examples/window_positioning.rs` | 246 | `examples/window_positioning/main.go` | 0 | 0 |
| `gpui/examples/window_shadow.rs` | 252 | `examples/window_shadow/main.go` | 0 | 0 |

### examples-d

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/examples/animation.rs` | 373 | `examples/animation/main.go` | 0 | 1 |
| `gpui/examples/input.rs` | 912 | `examples/input/main.go` | 0 | 3 |
| `gpui/examples/testing.rs` | 561 | `examples/testing/main.go` | 0 | 7 |
| `gpui/examples/view_example/example_editor.rs` | 541 | `examples/view_example/example_editor.go` | 0 | 0 |
| `gpui/examples/view_example/example_input.rs` | 121 | `examples/view_example/example_input.go` | 0 | 0 |
| `gpui/examples/view_example/example_tests.rs` | 131 | `examples/view_example/example_tests.go` | 0 | 4 |
| `gpui/examples/view_example/example_text_area.rs` | 118 | `examples/view_example/example_text_area.go` | 0 | 0 |
| `gpui/examples/view_example/view_example_main.rs` | 179 | `examples/view_example/view_example_main.go` | 0 | 0 |

### core-tests

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui/build.rs` | 47 | `-` | 0 | 0 |
| `gpui/benches/focus_cleanup.rs` | 141 | `bench_focus_cleanup_test.go` | 0 | 3 |
| `gpui/tests/action_macros.rs` | 55 | `integration_action_macros_test.go` | 0 | 1 |
| `gpui/tests/eq_hash.rs` | 36 | `integration_eq_hash_test.go` | 0 | 2 |
### platform-select

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_platform/benches/atlas_upload.rs` | 264 | `platform/bench_atlas_upload_test.go` | 0 | 0 |
| `gpui_platform/benches/inspector_render.rs` | 279 | `platform/bench_inspector_render_test.go` | 0 | 2 |
| `gpui_platform/benches/primitives.rs` | 443 | `platform/bench_primitives_test.go` | 0 | 1 |
| `gpui_platform/benches/randomized_element_tree.rs` | 1603 | `platform/bench_randomized_element_tree_test.go` | 0 | 24 |
| `gpui_platform/src/gpui_platform.rs` | 231 | `platform/platform.go` | 10 | 3 |

### linux-common

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_linux/src/linux/dispatcher.rs` | 369 | `platform/linux/internal/common/dispatcher.go` | 4 | 1 |
| `gpui_linux/src/linux/headless.rs` | 47 | `platform/linux/internal/common/headless.go` | 0 | 0 |
| `gpui_linux/src/linux/keyboard.rs` | 22 | `platform/linux/internal/common/keyboard.go` | 0 | 0 |
| `gpui_linux/src/linux/platform.rs` | 2223 | `platform/linux (+internal/common)/platform.go (+common.go)` | 1 | 16 |
| `gpui_linux/src/linux/text_system.rs` | 1 | `platform/linux/internal/common/text_system.go` | 0 | 0 |
| `gpui_linux/src/linux/headless/window.rs` | 208 | `platform/linux/internal/common/headless_window.go` | 0 | 0 |

### linux-dbus

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_linux/src/linux/power.rs` | 265 | `platform/linux/internal/dbus/power.go` | 0 | 2 |
| `gpui_linux/src/linux/system_notifications.rs` | 154 | `platform/linux/internal/dbus/system_notifications.go` | 0 | 1 |
| `gpui_linux/src/linux/xdg_desktop_portal.rs` | 249 | `platform/linux/internal/dbus/xdg_desktop_portal.go` | 3 | 1 |

### linux-platform

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_linux/src/gpui_linux.rs` | 4 | `platform/linux/linux.go` | 0 | 0 |
| `gpui_linux/src/linux.rs` | 54 | `platform/linux/linux.go` | 2 | 0 |
| `gpui_linux/src/linux/display_connection.rs` | 222 | `platform/linux/display_connection.go` | 0 | 0 |

### linux-wayland

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_linux/src/linux/wayland.rs` | 42 | `platform/linux/wayland/wayland.go` | 0 | 0 |
| `gpui_linux/src/linux/wayland/client.rs` | 3454 | `platform/linux/wayland/client.go` (1-1967), `client_input.go` (1968-3454) | 20 | 10 |
| `gpui_linux/src/linux/wayland/clipboard.rs` | 257 | `platform/linux/wayland/clipboard.go` | 0 | 0 |
| `gpui_linux/src/linux/wayland/cursor.rs` | 152 | `platform/linux/wayland/cursor.go` | 0 | 0 |
| `gpui_linux/src/linux/wayland/display.rs` | 77 | `platform/linux/wayland/display.go` | 0 | 1 |
| `gpui_linux/src/linux/wayland/layer_shell.rs` | 26 | `platform/linux/wayland/layer_shell.go` | 0 | 0 |
| `gpui_linux/src/linux/wayland/popup.rs` | 38 | `platform/linux/wayland/popup.go` | 0 | 0 |
| `gpui_linux/src/linux/wayland/serial.rs` | 169 | `platform/linux/wayland/serial.go` | 0 | 5 |
| `gpui_linux/src/linux/wayland/window.rs` | 2627 | `platform/linux/wayland/window.go` | 45 | 6 |

### linux-x11

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_linux/src/linux/x11.rs` | 13 | `platform/linux/x11/x11.go` | 0 | 0 |
| `gpui_linux/src/linux/x11/client.rs` | 3219 | `platform/linux/x11/client.go` | 8 | 11 |
| `gpui_linux/src/linux/x11/clipboard.rs` | 1287 | `platform/linux/x11/clipboard.go` | 2 | 0 |
| `gpui_linux/src/linux/x11/display.rs` | 56 | `platform/linux/x11/display.go` | 0 | 0 |
| `gpui_linux/src/linux/x11/event.rs` | 154 | `platform/linux/x11/event.go` | 0 | 1 |
| `gpui_linux/src/linux/x11/window.rs` | 2098 | `platform/linux/x11/window.go` | 3 | 0 |
| `gpui_linux/src/linux/x11/xim_handler.rs` | 133 | `platform/linux/x11/xim_handler.go` | 3 | 0 |
### wgpu

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_wgpu/src/gpui_wgpu.rs` | 15 | `render/wgpu/gpui_wgpu.go` | 0 | 0 |
| `gpui_wgpu/src/wgpu_atlas.rs` | 568 | `render/wgpu/wgpu_atlas.go` | 8 | 5 |
| `gpui_wgpu/src/wgpu_context.rs` | 848 | `render/wgpu/wgpu_context.go` | 21 | 3 |
| `gpui_wgpu/src/wgpu_renderer.rs` | 2900 | `render/wgpu/wgpu_renderer.go` | 23 | 10 |

### gotext

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_wgpu/benches/layout_line.rs` | 104 | `text/gotext/bench_layout_line_test.go` | 0 | 0 |
| `gpui_wgpu/src/cosmic_text_system.rs` | 1900 | `text/gotext/text_system.go` | 5 | 30 |
### apple

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_apple/build.rs` | 222 | `-` | 1 | 0 |
| `gpui_apple/src/dispatcher.rs` | 201 | `platform/apple/dispatcher.go` | 4 | 0 |
| `gpui_apple/src/display_link.rs` | 104 | `platform/apple/display_link.go` | 5 | 0 |
| `gpui_apple/src/gpui_apple.rs` | 47 | `platform/apple/gpui_apple.go` | 1 | 1 |
| `gpui_apple/src/keychain.rs` | 283 | `platform/apple/keychain.go` | 6 | 4 |
| `gpui_apple/src/metal_atlas.rs` | 572 | `platform/apple/metal_atlas.go` | 1 | 6 |
| `gpui_apple/src/metal_renderer.rs` | 1903 | `platform/apple/metal_renderer.go` | 26 | 3 |
| `gpui_apple/src/open_type.rs` | 171 | `platform/apple/open_type.go` | 1 | 0 |
| `gpui_apple/src/process_info.rs` | 182 | `platform/apple/process_info.go` | 5 | 1 |
| `gpui_apple/src/text_system.rs` | 1065 | `platform/apple/text_system.go` | 3 | 6 |

### macos

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_macos/src/display.rs` | 182 | `platform/macos/display.go` | 0 | 0 |
| `gpui_macos/src/display_link.rs` | 712 | `platform/macos/display_link.go` | 40 | 4 |
| `gpui_macos/src/events.rs` | 579 | `platform/macos/events.go` | 0 | 0 |
| `gpui_macos/src/gpui_macos.rs` | 126 | `platform/macos/gpui_macos.go` | 0 | 0 |
| `gpui_macos/src/keyboard.rs` | 1508 | `platform/macos/keyboard.go` | 0 | 0 |
| `gpui_macos/src/pasteboard.rs` | 512 | `platform/macos/pasteboard.go` | 15 | 6 |
| `gpui_macos/src/platform.rs` | 1680 | `platform/macos/platform.go` | 2 | 0 |
| `gpui_macos/src/screen_capture.rs` | 291 | `platform/macos/screen_capture.go` | 2 | 0 |
| `gpui_macos/src/system_notifications.rs` | 415 | `platform/macos/system_notifications.go` | 0 | 1 |
| `gpui_macos/src/window.rs` | 4344 | `platform/macos/window.go` | 1 | 1 |
| `gpui_macos/src/window_appearance.rs` | 36 | `platform/macos/window_appearance.go` | 0 | 0 |
| `gpui_macos/src/window/native_views.rs` | 1622 | `platform/macos/window_native_views.go` | 0 | 6 |

### ios

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_ios/examples/ios.rs` | 64 | `examples/ios_ios/main.go` | 0 | 0 |
| `gpui_ios/src/gpui_ios.rs` | 21 | `platform/ios/gpui_ios_impl.go` | 1 | 0 |
| `gpui_ios/src/ios.rs` | 58 | `platform/ios/ios.go` | 1 | 0 |
| `gpui_ios/src/ios/application.rs` | 543 | `platform/ios/application.go` | 0 | 6 |
| `gpui_ios/src/ios/display.rs` | 97 | `platform/ios/display.go` | 0 | 0 |
| `gpui_ios/src/ios/events.rs` | 128 | `platform/ios/events.go` | 6 | 0 |
| `gpui_ios/src/ios/platform.rs` | 519 | `platform/ios/platform.go` | 2 | 0 |
| `gpui_ios/src/ios/text_input.rs` | 1111 | `platform/ios/text_input.go` | 0 | 2 |
| `gpui_ios/src/ios/window.rs` | 1105 | `platform/ios/window.go` | 1 | 4 |
| `gpui_ios/tests/text_input.rs` | 59 | `platform/ios/integration_text_input_test.go` | 0 | 0 |

### windows

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_windows/build.rs` | 232 | `-` | 2 | 0 |
| `gpui_windows/src/clipboard.rs` | 402 | `platform/windows/clipboard.go` | 0 | 0 |
| `gpui_windows/src/destination_list.rs` | 207 | `platform/windows/destination_list.go` | 0 | 0 |
| `gpui_windows/src/dialog.rs` | 340 | `platform/windows/dialog.go` | 0 | 6 |
| `gpui_windows/src/direct_manipulation.rs` | 361 | `platform/windows/direct_manipulation.go` | 0 | 0 |
| `gpui_windows/src/direct_write.rs` | 2076 | `platform/windows/direct_write.go` | 0 | 4 |
| `gpui_windows/src/directx_atlas.rs` | 420 | `platform/windows/directx_atlas.go` | 0 | 1 |
| `gpui_windows/src/directx_devices.rs` | 194 | `platform/windows/directx_devices.go` | 0 | 0 |
| `gpui_windows/src/directx_renderer.rs` | 2083 | `platform/windows/directx_renderer.go` | 1 | 0 |
| `gpui_windows/src/dispatcher.rs` | 191 | `platform/windows/dispatcher.go` | 0 | 0 |
| `gpui_windows/src/display.rs` | 244 | `platform/windows/display.go` | 0 | 0 |
| `gpui_windows/src/events.rs` | 1844 | `platform/windows/events.go` | 0 | 0 |
| `gpui_windows/src/gpui_windows.rs` | 43 | `platform/windows/gpui_windows_impl.go` | 0 | 0 |
| `gpui_windows/src/keyboard.rs` | 371 | `platform/windows/keyboard.go` | 0 | 1 |
| `gpui_windows/src/platform.rs` | 2045 | `platform/windows/platform.go` | 2 | 4 |
| `gpui_windows/src/system_notifications.rs` | 198 | `platform/windows/system_notifications.go` | 0 | 0 |
| `gpui_windows/src/system_settings.rs` | 87 | `platform/windows/system_settings.go` | 0 | 0 |
| `gpui_windows/src/util.rs` | 191 | `platform/windows/util.go` | 0 | 0 |
| `gpui_windows/src/vsync.rs` | 119 | `platform/windows/vsync.go` | 0 | 2 |
| `gpui_windows/src/window.rs` | 1742 | `platform/windows/window.go` | 1 | 2 |
| `gpui_windows/src/wrapper.rs` | 53 | `platform/windows/wrapper.go` | 0 | 0 |

### web

| Upstream file | LOC | Go file | Public items | Tests |
|---|---|---|---|---|
| `gpui_web/examples/hello_web/dynamic_fonts.rs` | 130 | `examples/web_hello_web/dynamic_fonts.go` | 0 | 0 |
| `gpui_web/examples/hello_web/font_fallback_test.rs` | 117 | `examples/web_hello_web/font_fallback_testing.go` | 0 | 0 |
| `gpui_web/examples/hello_web/http_client_test.rs` | 50 | `examples/web_hello_web/http_client_testing.go` | 0 | 0 |
| `gpui_web/examples/hello_web/main.rs` | 464 | `examples/web_hello_web/main.go` | 0 | 0 |
| `gpui_web/examples/hello_web/queue_test.rs` | 122 | `examples/web_hello_web/queue_testing.go` | 0 | 0 |
| `gpui_web/src/canvas_fallback.rs` | 490 | `platform/web/canvas_fallback.go` | 3 | 7 |
| `gpui_web/src/canvas_text.rs` | 197 | `platform/web/canvas_text.go` | 0 | 0 |
| `gpui_web/src/dispatcher.rs` | 425 | `platform/web/dispatcher.go` | 2 | 0 |
| `gpui_web/src/display.rs` | 107 | `platform/web/display.go` | 2 | 0 |
| `gpui_web/src/events.rs` | 1516 | `platform/web/events.go` | 1 | 2 |
| `gpui_web/src/glyph_cache.rs` | 133 | `platform/web/glyph_cache.go` | 0 | 3 |
| `gpui_web/src/gpui_web.rs` | 54 | `platform/web/gpui_web.go` | 0 | 0 |
| `gpui_web/src/http_client.rs` | 312 | `platform/web/http_client.go` | 5 | 0 |
| `gpui_web/src/ime_mirror.rs` | 618 | `platform/web/ime_mirror.go` | 0 | 0 |
| `gpui_web/src/keyboard.rs` | 13 | `platform/web/keyboard.go` | 1 | 0 |
| `gpui_web/src/logging.rs` | 45 | `platform/web/logging.go` | 1 | 0 |
| `gpui_web/src/platform.rs` | 846 | `platform/web/platform.go` | 7 | 0 |
| `gpui_web/src/run_replacements.rs` | 594 | `platform/web/run_replacements.go` | 0 | 9 |
| `gpui_web/src/text_system.rs` | 509 | `platform/web/text_system.go` | 0 | 0 |
| `gpui_web/src/viewport.rs` | 268 | `platform/web/viewport.go` | 0 | 0 |
| `gpui_web/src/window.rs` | 1054 | `platform/web/window.go` | 1 | 0 |

## 5. Deviations and limits

A row may leave `todo` only as `done`, `dev:<id>` or `n/a:<id>` with an id from these tables, or temporarily as `blocked:<stage>` while it waits for a later stage. `blocked:owner` is not used. New ids are registered by the stage that needs them in a `## Deviations registered here` table at the end of its own area file, with the upstream location and the reason; exit stages fold them into this section.

A row that uses a limit cites it as `dev:L-<x>` when the behaviour differs and `n/a:L-<x>` when the item cannot exist in Go.

Known parity limits (behaviour differs):

| Id | Limit | Upstream | Source |
|---|---|---|---|
| L-GLYPH | Linux glyph rasterisation differs from swash | `gpui_wgpu/src/cosmic_text_system.rs:18`, `:52` | RESEARCH-deps.md |
| L-JPEG | JPEG decode may differ in pixel values | `gpui/src/platform.rs:3462-3486` | RESEARCH-deps.md |
| L-SVGTEXT | text and embedded images inside SVG | `gpui/src/svg_renderer.rs:369-517` | RESEARCH-deps.md |
| L-WGPUVER | the binding is wgpu-native v29 (oliverbestmann/webgpu v1.38.1) while upstream targets wgpu 30; where v29 lacks `EnumerateAdapters`, `DUAL_SOURCE_BLENDING` or `TRANSIENT_ATTACHMENT`, M1-wgpu-context's fallbacks apply (RequestAdapter with the compositor hint, the non-subpixel path, plain RENDER_ATTACHMENT) | `gpui_wgpu/src/wgpu_context.rs:215`, `:531`, `:442-446`, `gpui_wgpu/src/wgpu_renderer.rs:1571` | owner decision (DESIGN.md 14.3); TECH_DEBT.md tracks bumping the binding |
| L-WEBTHREADS | Go wasm has no threads, so the multi-threaded web dispatcher mode is absent; shared paths follow the single-threaded branch | `gpui_web/src/platform.rs:147-156`, `gpui_web/src/dispatcher.rs` | owner decision (DESIGN.md 14.3) |

Intentional deviations (behaviour kept, form changed):

| Id | Deviation | Upstream evidence | DESIGN |
|---|---|---|---|
| D-NAME | Constructor functions that collide with a type become `New<Type>`; `Element::id` and `Element::a11y_synthetic_children` are `ElementID` and `ElementA11ySyntheticChildren`; digit boundaries keep `_` | `gpui/src/elements/div.rs:1844` (`div()` next to `Div` at `:1905`), `gpui/src/element.rs:66`, `gpui/src/elements/div.rs:803` | 5.1, 6.5 |
| D-RAII | No `Drop`: `Release`, `Cancel`, `Unsubscribe` are explicit, with a GC cleanup as safety net for handles only | `gpui/src/app/entity_map.rs:440-466`, `gpui/src/window.rs:674`, `gpui/src/subscription.rs:300`, `scheduler/src/executor.rs:437` | 5.2 |
| D-OPT | `Option` and `Result` become `(T, bool)`, `(T, error)`, nil-able types, or `Option[T]` fields | `gpui/src/style.rs:240` | 5.3 |
| D-VOID | Closure APIs generic over a return type get a void form and an `R`-suffixed generic form | `gpui/src/app/entity_map.rs:577-585` | 5.4 |
| D-EMIT | `EventEmitter<E>` is not enforced at compile time | `gpui/src/gpui.rs:299`, `gpui/src/app/context.rs:93-108` | 5.4 |
| D-COND | Conditional trait impls are always-present methods with a run-time check | `gpui/src/elements/div.rs:4281`, `gpui/src/view.rs:112` | 5.4 |
| D-INTO | `impl Into<X>` parameters are concrete types, sealed `IntoX` interfaces, or union-constrained generic methods | `gpui_macros/src/styles.rs:689`, `gpui/src/elements/div.rs:803` | 5.4 |
| D-GEOM | Geometry containers split into a numeric family with arithmetic and an `...Of[T]` family; impls on one instantiation become free functions | `gpui/src/geometry.rs:86`, `:397`, `:724`, `:1785`, `:2349`, `gpui/src/style.rs:190` | 6.1 |
| D-STRING | `SharedString` is a Go string type; `ArcCow` is not ported | `gpui_shared_string/gpui_shared_string.rs:15`, `gpui_util/src/arc_cow.rs:9` | 5.5 |
| D-ASYNC | Futures become coroutine tasks; `.await` is `Await(cx)`; `Task<T>` and `Task<Result<T>>` are both `*Task[T]` returning `(T, error)`; app access from tasks goes through `AsyncApp.Update` | `scheduler/src/scheduler.rs:100-135`, `gpui/src/app/async_context.rs:22-34` | 5.6, 6.3 |
| D-RNG | Seeded PCG instead of Rust `StdRng`: same determinism, different interleavings per seed | `scheduler/src/test_scheduler.rs:904` | 5.6 |
| D-ARENA | Element arena replaced by the GC and frame pools | `gpui/src/arena.rs:81`, `:218` | 5.7 |
| D-TOKIO | `gpui_tokio` not ported | `gpui_tokio/src/gpui_tokio.rs:12-97` | 5.7 |
| D-HOTRELOAD | subsecond hot reload not ported | `gpui/src/gpui.rs:69-70`, `gpui/src/hot_reload.rs` | 5.7 |
| D-PROPTEST | Property tests run seeded generators without shrinking | `gpui/src/test.rs:43-66`, `sum_tree/src/property_test.rs` | 5.7 |
| D-BENCH | Criterion benches become `testing.B` benchmarks | `gpui_macros/src/gpui_macros.rs:222`, `gpui/src/bench.rs:18-51` | 5.7 |
| D-SCHEMA | Action JSON schema from an in-repo reflector instead of schemars | `gpui/src/action.rs:139` | 5.7 |
| D-FEATURE | Cargo features become always-compiled code, run-time switches, or the build tag `gpui_profiler` | `gpui/Cargo.toml:20-57` | 5.7 |
| D-HTTP | `http_client` crate becomes the `HTTPClient` interface over `net/http` | `gpui/src/gpui.rs:6`, `gpui/src/app.rs:3747` | 5.7 |
| D-MACRO | Proc macros become generators, embedding, reflection or helpers | `gpui_macros/src/gpui_macros.rs:20-317` | 9 |

Owner decisions (DESIGN.md section 14, all decided): the WebGL2 fallback and screen capture (Linux portal + PipeWire and X11, Windows Graphics Capture, macOS ScreenCaptureKit) are ported, not limits; the wgpu version gap and web threading are the accepted limits L-WGPUVER and L-WEBTHREADS above. No other `L-` id may be added without the owner.

## 6. Examples

Every program under `gpui/examples`, plus the web and iOS example crates. Linux column: how it is verified locally. All start as `todo`.

| Upstream | LOC | Go package | Notes | Status |
|---|---|---|---|---|
| `gpui/examples/a11y.rs` | 272 | `examples/a11y` | needs an a11y adapter to be observable | todo |
| `gpui/examples/active_state_bug.rs` | 47 | `examples/active_state_bug` |  | todo |
| `gpui/examples/anchor.rs` | 207 | `examples/anchor` |  | todo |
| `gpui/examples/animation.rs` | 373 | `examples/animation` |  | todo |
| `gpui/examples/data_table.rs` | 494 | `examples/data_table` |  | todo |
| `gpui/examples/drag_drop.rs` | 158 | `examples/drag_drop` |  | todo |
| `gpui/examples/focus_visible.rs` | 235 | `examples/focus_visible` |  | todo |
| `gpui/examples/gif_viewer.rs` | 62 | `examples/gif_viewer` |  | todo |
| `gpui/examples/gradient.rs` | 278 | `examples/gradient` |  | todo |
| `gpui/examples/grid_layout.rs` | 95 | `examples/grid_layout` |  | todo |
| `gpui/examples/hello_world.rs` | 127 | `examples/hello_world` |  | todo |
| `gpui/examples/image_gallery.rs` | 300 | `examples/image_gallery` |  | todo |
| `gpui/examples/image_loading.rs` | 232 | `examples/image_loading` |  | todo |
| `gpui/examples/input.rs` | 912 | `examples/input` |  | todo |
| `gpui/examples/layer_shell.rs` | 101 | `examples/layer_shell` | Linux Wayland only (`cfg(all(target_os = "linux", feature = "wayland"))`) | todo |
| `gpui/examples/list_example.rs` | 176 | `examples/list_example` |  | todo |
| `gpui/examples/mouse_pressure.rs` | 87 | `examples/mouse_pressure` |  | todo |
| `gpui/examples/move_entity_between_windows.rs` | 160 | `examples/move_entity_between_windows` |  | todo |
| `gpui/examples/native_view.rs` | 571 | `examples/native_view` | macOS only (`cfg(target_os = "macos")`); other OSes print and exit as upstream | todo |
| `gpui/examples/native_view_appkit_field.rs` | 226 | `examples/native_view_appkit_field` | macOS only | todo |
| `gpui/examples/on_window_close_quit.rs` | 103 | `examples/on_window_close_quit` |  | todo |
| `gpui/examples/opacity.rs` | 195 | `examples/opacity` |  | todo |
| `gpui/examples/ownership_post.rs` | 56 | `examples/ownership_post` |  | todo |
| `gpui/examples/painting.rs` | 533 | `examples/painting` |  | todo |
| `gpui/examples/paths_bench.rs` | 113 | `examples/paths_bench` | benchmark-style example | todo |
| `gpui/examples/pattern.rs` | 136 | `examples/pattern` |  | todo |
| `gpui/examples/popover.rs` | 197 | `examples/popover` |  | todo |
| `gpui/examples/scrollable.rs` | 78 | `examples/scrollable` |  | todo |
| `gpui/examples/set_menus.rs` | 134 | `examples/set_menus` | app menus; visible on macOS | todo |
| `gpui/examples/shadow.rs` | 622 | `examples/shadow` |  | todo |
| `gpui/examples/svg_animation.rs` | 222 | `examples/svg_animation` |  | todo |
| `gpui/examples/system_notifications.rs` | 155 | `examples/system_notifications` | needs the notification back end | todo |
| `gpui/examples/tab_stop.rs` | 220 | `examples/tab_stop` |  | todo |
| `gpui/examples/testing.rs` | 561 | `examples/testing` | contains tests; ported with its tests | todo |
| `gpui/examples/text.rs` | 432 | `examples/text` |  | todo |
| `gpui/examples/text_layout.rs` | 117 | `examples/text_layout` |  | todo |
| `gpui/examples/text_wrapper.rs` | 144 | `examples/text_wrapper` |  | todo |
| `gpui/examples/toggle_headless.rs` | 309 | `examples/toggle_headless` | windowing-mode switch; behaviour differs per OS as upstream | todo |
| `gpui/examples/tree.rs` | 64 | `examples/tree` |  | todo |
| `gpui/examples/uniform_list.rs` | 71 | `examples/uniform_list` |  | todo |
| `gpui/examples/window.rs` | 355 | `examples/window` |  | todo |
| `gpui/examples/window_movable.rs` | 131 | `examples/window_movable` |  | todo |
| `gpui/examples/window_positioning.rs` | 246 | `examples/window_positioning` |  | todo |
| `gpui/examples/window_shadow.rs` | 252 | `examples/window_shadow` |  | todo |
| `gpui/examples/image/` | 220 | `examples/image_example` | ships 5 asset files | todo |
| `gpui/examples/svg/` | 108 | `examples/svg_example` | ships `dragon.svg` | todo |
| `gpui/examples/view_example/` | 1090 | `examples/view_example` | 5 files, contains tests | todo |
| `gpui/examples/example_support/fonts.rs` | 44 | `examples/internal/support` | shared helper, not a program | todo |
| `gpui_web/examples/hello_web/` | 883 | `examples/web_hello_web` | GOOS=js; 5 Rust files plus html and js test drivers | todo |
| `gpui_ios/examples/ios.rs` | 64 | `examples/ios_ios` | GOOS=ios; Xcode wrapper in `gpui_ios/examples/ios/` | todo |

47 example programs in `gpui/examples`.

## 7. External code ported in-repo, and assets

These are not in `_ref/` (they are dependencies of the crates above) but their behaviour is part of parity. The owning stage enumerates their tests from the pinned version and adds an area file.

| Dependency | Version | Go package | Tests to port | Area file to create |
|---|---|---|---|---|
| taffy | =0.15.0 (`../Cargo.toml:932`; features `gpui/Cargo.toml:108-116`) | `internal/taffy` | the generated fixture tests for flexbox, grid, block and float, plus unit tests | `parity/ext-taffy.md` |
| etagere | 0.2 (`../Cargo.toml:638`) | `internal/etagere` | its unit tests for `BucketedAtlasAllocator` | `parity/ext-etagere.md` |
| lyon_path, tiny-skia-path | 1.0.19, 0.11 (`gpui/Cargo.toml`) | `internal/lyon` | tests for the subset `gpui/src/path_builder.rs:3-6` uses | `parity/ext-lyon.md` |
| accesskit (schema only) | 0.24.1 (`../Cargo.toml:539`) | `a11y` | none upstream in scope; schema equality checks | `parity/ext-a11y.md` |
| accesskit_consumer | 0.38.0 | `a11y/consumer` | its unit tests | `parity/ext-a11y-consumer.md` |
| accesskit_atspi_common, accesskit_unix | 0.19.1, 0.22.1 | `a11y/atspi` | their unit tests; bus tests against a private dbus-daemon | `parity/ext-a11y-atspi.md` |
| accesskit_macos | 0.26.3 | `a11y/macos` | its unit tests (CI) | `parity/ext-a11y-macos.md` |
| accesskit_windows | 0.34.0 | `a11y/windows` | its unit tests and `tests/` (CI) | `parity/ext-a11y-windows.md` |
| zed-scap (Linux engines: portal + PipeWire, X11) | 0.0.8-zed (the rev `../Cargo.toml:898` pins) | `platform/linux/internal/screencap`, `internal/linux/pw` | none upstream; binding layout tests, gated capture tests | `parity/ext-scap-linux.md` |
| zed-scap (Windows engine), windows-capture | 0.0.8-zed, 1.5.0 | `internal/win/wgc`, `internal/win/screencap` | none upstream; COM layout tests (CI) | `parity/ext-scap-windows.md` |
| zeno (mask rasteriser incl. `CustomSubpixel`), swash (colour glyph sources) | 0.3.3, 0.2.10 | `text/gotext` | rows added under their own headings in `parity/gotext.md` | - |
| cbindgen header for Metal (`gpui_apple/build.rs:28-120`) | 0.28.0 | `internal/cmd/metalhdr` -> `platform/apple/shaders_scene_gen.go` | offset test against the Go scene structs | rows in `parity/apple.md` |
| wgpu GL backend (naga WGSL -> GLSL ES 3.00, wgpu-hal GLES) | wgpu rev cddb2f5 (`../Cargo.toml:1013`), wgpu-hal 30.0.1, naga 30.0.1 | `internal/cmd/glslgen`, `internal/webglsl`, `render/wgpu/internal/gpu/webgl2` | binding-table and state tests under node | rows in `parity/wgpu.md` |

Non-Rust upstream files that ship with the port:

| Upstream | Lines | Go use |
|---|---|---|
| `gpui_wgpu/src/shaders.wgsl` | 1507 | `//go:embed` in `render/wgpu`, byte-identical |
| `gpui_wgpu/src/shaders_storage.wgsl` | 48 | `//go:embed` in `render/wgpu`, byte-identical |
| `gpui_wgpu/src/shaders_subpixel.wgsl` | 27 | `//go:embed` in `render/wgpu`, byte-identical |
| `gpui_wgpu/src/shaders_webgl.wgsl` | 286 | `//go:embed` in `render/wgpu`, byte-identical |
| `gpui_apple/src/shaders.metal` | 1354 | `//go:embed` in `platform/apple`, compiled at start after the cbindgen scene.h header |
| `gpui_windows/src/shaders.hlsl` | 1372 | `//go:embed` in `platform/windows`, compiled at start; its `#include "alpha_correction.hlsl"` is served by an ID3DInclude |
| `gpui_windows/src/alpha_correction.hlsl` | 49 | `//go:embed` in `platform/windows`, compiled at start |
| `gpui/examples/image/*` (png, svg, gif, jpg), `gpui/examples/svg/dragon.svg` | - | copied into the Go example directories |
| `gpui/docs/contexts.md`, `gpui/docs/key_dispatch.md` | - | rewritten for the Go API under `docs/` |
| `gpui/build.rs`, `gpui_apple/build.rs`, `gpui_windows/build.rs` | - | no Go file; their work moves to run time or `go generate` (DESIGN.md 8.3, 8.4) |

## 8. Workflow

1. A stage owns the area files listed against its Go paths in section 2 and edits only those.
2. When an item is ported, its row becomes `done`. If the Go name differs from the rule-derived one, the Go column is edited and the status is `dev:D-NAME`.
3. When a test is ported, its row becomes `done`; the Go test must exist under the listed name. Tests gated by `cfg(target_os)` keep the gate as a Go build constraint; `#[ignore]` becomes `t.Skip` with the upstream reason.
4. An item that cannot exist in Go becomes `n/a:<id>` or `dev:<id>`; a new id is registered in the stage's own area file (section 5) in the same change. A row that needs a later stage is `blocked:<stage>`; `paritycheck -all-done -allow-blocked` tolerates it until that stage is done.
5. A stage that finds a public item or test the scan missed adds the row.
6. `go run ./internal/cmd/paritycheck` (built in M0-bootstrap) must pass before a stage is complete.

## 9. How the matrix was generated

A one-off scanner walked the 300 `.rs` files of the 17 crates at the pinned commit. M0-parity-rescan reimplements it as `internal/cmd/paritycheck -rescan`, so the rules are stated here.

- Comments and string contents are blanked; brace depth tracks `mod`, `impl`, `trait` and function bodies.
- A public item is: a `pub` struct, enum, union, type alias, const, static or free fn at module level; a `pub fn` or `pub const` in an inherent impl whose type is not declared crate-private; every fn of a `pub trait`; a `#[macro_export]` macro; a `#[proc_macro*]` function. `pub(crate)` and narrower are not public. For `gpui_macros` only the proc macros count.
- Items under `#[cfg(test)]`, in `mod tests`, or in files named `tests.rs` / `*_tests.rs` / under `tests/`, `benches/`, `examples/` are not public items; functions there with a test attribute are tests.
- A test is a fn under `#[test]`, `#[gpui::test]`, `#[crate::test]`, `#[gpui::property_test]`, `#[perf]` (the `util_macros::perf` test wrapper, `gpui/src/style.rs:1339`) or `#[gpui::bench]`, with any arguments.
- The `Cfg` column lists `target_os`, `target_family` and `feature` names found on the item or its enclosing blocks.
- Style macro methods are expanded from the prefix and suffix tables in `gpui_macros/src/styles.rs:699-1417` with the rules of `generate_box_style_methods`, `generate_methods` and `border_style_methods` (`:531-621`, `:335-360`).

Known blind spots, to be closed by the owning stage reading the file: items produced by other macros (for example `slotmap::new_key_type!` is handled, `bitflags!` and `actions!` invocations are not expanded), tests declared inside macro invocations, enum variants and struct fields (covered by rule N12 and by the type row), and trait impls (which types implement which traits is in DESIGN.md 6.5 and the upstream `impl` lines).

Syntactically public items of crate-private modules are listed too (for example parts of `gpui_linux`); the owning stage may port them unexported and mark the row `done`.
