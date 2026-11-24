// Command system_notifications demonstrates posting, replacing, dismissing, and responding to
// system notifications.
package main

import (
	"fmt"
	"runtime"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

const notificationTag = "gpui-system-notification-example"

// SystemNotificationExample posts and dismisses a tagged notification.
type SystemNotificationExample struct {
	revision int
	status   gpui.SharedString
}

// Render renders the example.
func (s *SystemNotificationExample) Render(_ *gpui.Window, cx *gpui.Context[SystemNotificationExample]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		SizeFull().
		Gap4().
		P8().
		Bg(gpui.Rgb(0x18181b)).
		TextColor(gpui.Rgb(0xf4f4f5)).
		Child(gpui.NewDiv().Text2xl().ChildText("GPUI system notifications")).
		Child(gpui.NewDiv().TextSm().TextColor(gpui.Rgb(0xa1a1aa)).ChildText(
			"Post repeatedly to replace the notification with the same tag. Click the "+
				"notification body or an action button to send a response back to GPUI.",
		)).
		Child(
			gpui.NewDiv().
				Flex().
				Gap3().
				Child(button("show", "Show or replace").OnClick(cx.Listener[gpui.ClickEvent](
					func(this *SystemNotificationExample, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[SystemNotificationExample]) {
						this.revision++
						revision := this.revision
						cx.ShowSystemNotification(gpui.SystemNotification{
							Tag:   notificationTag,
							Title: gpui.SharedString(fmt.Sprintf("Example notification %d", revision)),
							Body:  "This notification was posted by the GPUI example.",
							Actions: []gpui.SystemNotificationAction{
								{ID: "open", Label: "Open"},
								{ID: "snooze", Label: "Snooze"},
							},
						})
						this.status = gpui.SharedString(fmt.Sprintf("Posted notification revision %d", revision))
						cx.Notify()
					},
				))).
				Child(
					button("dismiss", "Dismiss").OnClick(cx.Listener[gpui.ClickEvent](
						func(this *SystemNotificationExample, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[SystemNotificationExample]) {
							cx.DismissSystemNotification(notificationTag).Detach()
							this.status = "Dismissed the notification"
							cx.Notify()
						},
					)),
				),
		).
		Child(
			gpui.NewDiv().
				Mt2().
				P4().
				RoundedMd().
				Bg(gpui.Rgb(0x27272a)).
				Child(s.status),
		).
		When(runtime.GOOS == "darwin", func(this *gpui.Div) *gpui.Div {
			return this.Child(gpui.NewDiv().Mt2().TextXs().TextColor(gpui.Rgb(0x71717a)).ChildText(
				"macOS only delivers notifications when this example runs from an app bundle.",
			))
		})
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func button(id string, label string) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(id).
		Px4().
		Py2().
		RoundedMd().
		Bg(gpui.Rgb(0x3f3f46)).
		Hover(func(style *gpui.StyleRefinement) *gpui.StyleRefinement { return styled(style).Bg(gpui.Rgb(0x52525b)) }).
		Active(func(style *gpui.StyleRefinement) *gpui.StyleRefinement { return styled(style).Bg(gpui.Rgb(0x71717a)) }).
		CursorPointer().
		ChildText(label)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		cx.SetAppIdentity("dev.zed.gpui.system-notifications", "GPUI Notifications")

		view := cx.New(func(_ *gpui.Context[SystemNotificationExample]) *SystemNotificationExample {
			return &SystemNotificationExample{
				revision: 0,
				status:   "No notification posted yet",
			}
		})
		{
			view := view.Clone()
			cx.OnSystemNotificationResponse(func(response gpui.SystemNotificationResponse, cx *gpui.App) {
				tag := response.Tag
				view.Update(cx, func(this *SystemNotificationExample, cx *gpui.Context[SystemNotificationExample]) {
					if actionID, ok := response.ActionID.Get(); ok {
						this.status = gpui.SharedString(fmt.Sprintf("Received action '%s' for tag '%s'", actionID, tag))
					} else {
						this.status = gpui.SharedString(fmt.Sprintf("Notification body clicked for tag '%s'", tag))
					}
					cx.Notify()
				})
			})
		}

		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(560.), gpui.Px(360.)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		options.Titlebar = gpui.Some(gpui.TitlebarOptions{
			Title: gpui.Some[gpui.SharedString]("System Notifications Example"),
		})
		if _, err := cx.OpenWindow(options, func(*gpui.Window, *gpui.App) gpui.Entity[SystemNotificationExample] {
			return view
		}); err != nil {
			panic(fmt.Sprintf("failed to open system notifications example window: %v", err))
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
