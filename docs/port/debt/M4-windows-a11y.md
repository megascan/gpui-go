# M4-windows-a11y debt

- platform/windows never calls `A11yState.release()` on window destroy (window.go / events.go are outside this stage); handed off in docs/port/handoff/M4-windows-a11y.md. Effect: a destroyed window's UIA platform nodes stay referenced.
- The Windows UIA tests have only been cross-compiled here; first real run is the Windows CI job.
