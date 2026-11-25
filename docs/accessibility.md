# Accessibility in GPUI

"Accessibility" means that everyone can use your application, with or without a disability. It has many parts, all of them important, for example:

- Sufficient text contrast.
- A way to turn off animations.
- A way to make text larger.

This guide covers **programmatic accessibility**: letting assistive technology, such as screen readers and Braille displays, inspect and operate your app. Contributors working on the accessibility support itself should read the docs of package `a11y`.

GPUI's accessibility tree follows [AccessKit]'s model; package `a11y` is a Go port of it, and the platform adapters (AT-SPI on Linux, NSAccessibility on macOS, UI Automation on Windows) are native Go ports of AccessKit's. The rest of this guide calls programmatic accessibility simply "accessibility".

A minimal example is in `examples/a11y`.

## Background

Accessibility rests on two capabilities:

- Telling assistive technology what the UI currently shows.
- Carrying out actions that assistive technology requests.

For example, a screen reader may announce that a new button has appeared, and the user may then press it with a voice control program.

### IDs in GPUI: `ElementID` and `GlobalElementID`

Each element can have an ID:

```go
divWithID := gpui.NewDiv().ID("my-id").Child(gpui.TextMacro("hello"))

// IDs are optional
divWithoutID := gpui.NewDiv().Child(gpui.TextMacro("hello"))
```

An element with an ID also gets a `GlobalElementID`, made of the IDs of all its ancestors that have one. For example:

```go
inner := gpui.NewDiv().ID("inner-id")
middle := gpui.NewDiv().Child(inner) // no ID
outer := gpui.NewDiv().ID("outer-id").Child(middle)
```

Here the global ID of `inner` is, roughly, `["outer-id", "inner-id"]`. `middle` has no ID, so it has no global ID either.

Global IDs should be unique within a frame. Two elements with the same global ID in one frame will probably cause bugs.

### IDs and accessibility

When GPUI renders a frame, it walks the element tree, finds the elements with global IDs and reports them to assistive technology.

To be reported, an element also needs a role, which tells assistive technology what kind of node it is (button, label, table, ...). Set it with `Role` after `ID`: `gpui.NewDiv().ID(...).Role(a11y.RoleButton)`. A custom element reports its role by implementing `gpui.ElementA11y`.

Nodes with the same global ID in different frames are treated as the same node. For example:

```go
// The UI in frame 1
frame1 := gpui.NewDiv().
	ID("parent").
	Role(a11y.RoleButton).
	Child(
		gpui.NewDiv().
			ID("id-1").
			Role(a11y.RoleLabel).
			Child(gpui.TextMacro("hello")),
	)

// The UI in the next frame
frame2 := gpui.NewDiv().
	ID("parent").
	Role(a11y.RoleButton).
	Child(
		gpui.NewDiv().
			ID("id-2"). // <- different ID
			Role(a11y.RoleLabel).
			Child(gpui.TextMacro("hello")),
	)
```

Nothing changed for the user, but a screen reader cannot tell that the two child divs are the same, so it sees one node removed and another added. That is disorienting, because announcements usually mean something *meaningful* changed.

So the ID of an element decides whether a change to it counts as meaningful, and the role decides whether the element is reported at all: nodes without a role are left out.

#### IDs and text

Text needs care.

`gpui.TextMacro` (upstream's `text!` macro) wraps a string in a `Text` element and gives it an ID automatically. That is usually what you want, but the way the ID is chosen can be surprising.

The ID comes from **the place in the source code where `TextMacro` is called**. For example:

```go
a := gpui.TextMacro("a")
b := gpui.TextMacro("b")

// Different call sites, different IDs
idA, _ := a.ElementID()
idB, _ := b.ElementID()
// idA != idB

// However:

makeText := func(s gpui.SharedString) gpui.Text { return gpui.TextMacro(s) }

a = makeText("a")
b = makeText("b")

// Both come from the same TextMacro call, so their IDs are equal
```

This leads to a common mistake:

```go
todos := []gpui.SharedString{"eat lunch", "drink water", "go to gym"}
var todoDivs []gpui.IntoElement
for _, todo := range todos {
	todoDivs = append(todoDivs, gpui.TextMacro(todo))
}

gpui.NewDiv().
	ID("todo-list").
	Role(a11y.RoleDocument).
	Children(todoDivs...) // WRONG: several nodes with the same global ID
```

`TextMacro` is called from one place, so every item gets the same ID, and with the same ancestors they all get the same global ID. Some of the nodes will be silently dropped.

To fix it, give each one an ID:

```go
for index, todo := range todos {
	todoDivs = append(todoDivs, gpui.TextMacro(todo).WithID(gpui.NewElementID(index))) // or gpui.NewText(id, todo)
}
```

Or wrap each `TextMacro` in an element that *does* have a unique global ID:

```go
for index, todo := range todos {
	todoDivs = append(todoDivs, gpui.NewDiv().ID(index).Child(gpui.TextMacro(todo)))
}
```

This works too, because the accessibility node ID (`a11y.NodeID`) is derived from the global ID, which includes the IDs of all ancestors.

Sometimes you need a `Text` element with *no* ID; create it with `gpui.NewTextInaccessible`. This is useful in custom components such as buttons, where you set a label on the parent div and do not want the text to appear twice in the accessibility tree.

### Handling actions

Assistive technology can send actions to the UI. Many of its users use ordinary input devices such as a keyboard, but some use other systems: people with limited mobility may control the app by voice, for example.

An action is sent *to a specific node*, and each element decides how it responds.

These actions have **nothing to do** with GPUI's `Action` interface. They are AccessKit's actions, `a11y.Action`, also available as `gpui.AccessibleAction`.

To respond to one, use `OnA11yAction`:

```go
gpui.NewDiv().
	ID("my-slider").
	Role(a11y.RoleSlider).
	OnA11yAction(a11y.ActionIncrement, func(_ a11y.ActionData, _ *gpui.Window, cx *gpui.App) {
		_ = this.Update(cx, func(slider *Slider, cx *gpui.Context[Slider]) {
			slider.position += 1
			cx.Notify()
		})
	}).
	Child(myCoolSlider())
```

Some common actions are registered for you. `OnClick`, for example, also adds an `a11y.ActionClick` handler that runs the click handler.

## Synthetic children

A custom element may want to look like several nodes. A custom text editor element, say, may have `a11y.RoleTextInput` and present its content as children with `a11y.RoleTextRun`.

Implement `gpui.ElementA11ySyntheticChildren[PP]` on the element, where `PP` is its prepaint state type:

```go
func (f *MyCustomTextField) A11yRole() (a11y.Role, bool) {
	return a11y.RoleTextInput, true
}

func (f *MyCustomTextField) WriteA11yInfo(node *a11y.Node) {}

func (f *MyCustomTextField) ElementA11ySyntheticChildren(prepaint *myPrepaintState, builder *gpui.A11ySubtreeBuilder) {
	run := a11y.NewNode(a11y.RoleTextRun)
	run.SetValue(f.text)
	lengths := make([]uint8, 0, len(f.text))
	for _, r := range f.text {
		lengths = append(lengths, uint8(utf8.RuneLen(r)))
	}
	run.SetCharacterLengths(lengths)

	runID := builder.SyntheticNodeID(0)
	builder.PushChild(runID, run)

	caret := a11y.TextPosition{Node: runID, CharacterIndex: f.cursor}
	builder.ParentNode().SetTextSelection(a11y.TextSelection{Anchor: caret, Focus: caret})
}
```

A `Div` can add synthetic children without a custom element through `A11ySyntheticChildren(func(builder *gpui.A11ySubtreeBuilder) {...})`.

Synthetic children are added *after* the element is prepainted, so the hook can use the prepaint state, for example to find out what is visible.

## Further reading

Building accessible interfaces well takes the same care as building any interface well. These pages help:

- [AccessKit]: the cross-platform accessibility toolkit whose model GPUI follows.
- [MDN WAI-ARIA basics][mdn-aria]: an introduction to roles, properties and states.
- [ARIA Authoring Practices Guide][apg]: W3C patterns for accessible widgets.

GPUI borrows from web APIs, but it does not always behave *exactly* as a browser would with the same attributes.

[AccessKit]: https://accesskit.dev/
[mdn-aria]: https://developer.mozilla.org/en-US/docs/Learn_web_development/Core/Accessibility/WAI-ARIA_basics
[apg]: https://www.w3.org/WAI/ARIA/apg/
