# M4-windows-a11y-node owner checks

- YOU: on Windows x64, once M4-windows-a11y wires the adapter, hover a gpui example with Inspect.exe or Accessibility Insights and check the hovered element is found (IRawElementProviderFragmentRoot::ElementProviderFromPoint through the D-WINFLOAT trampoline).
- YOU: on Windows x64, set a slider's value through Inspect.exe's RangeValue pattern (IRangeValueProvider::SetValue) and check the example receives the new value.
- YOU: on an ARM64 Windows device (CI has no arm64 runner), repeat both checks and also TextPattern.RangeFromPoint on a text input (UiaPoint is passed in D0/D1 there).
