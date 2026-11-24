//go:build windows

package windows

import (
	"fmt"
	"log/slog"
	"math/bits"
	"unsafe"

	xwin "golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/scheduler"
)

var (
	iidIXmlDocument                     = com.MustGUID("{F7F3A506-1E87-42D6-BCFB-B8C809FA5494}")
	iidIXmlNode                         = com.MustGUID("{1C741D59-2122-47D5-A856-83F3D4214875}")
	iidIXmlNodeSerializer               = com.MustGUID("{5CC5B382-E6DD-4991-ABEF-06D8D2E7BD0C}")
	iidIToastNotification2              = com.MustGUID("{9DFB9FD1-143A-490E-90BF-B9FBA7132DE7}")
	iidIToastNotificationFactory        = com.MustGUID("{04124B20-82C6-4229-B109-FD9ED4662B53}")
	iidIToastNotificationManagerStatics = com.MustGUID("{50AC103F-D235-4598-BBEF-98FE4D1A3AD4}")
	iidIToastActivatedEventArgs         = com.MustGUID("{E3BF92F3-C197-436F-8265-0625824F8DAC}")
	iidToastActivatedHandler            = com.MustGUID("{AB54DE2D-97D9-5528-B6AD-105AFE156530}")
)

const (
	xmlDocumentClass               = "Windows.Data.Xml.Dom.XmlDocument"
	toastNotificationClass         = "Windows.UI.Notifications.ToastNotification"
	toastNotificationManagerClass  = "Windows.UI.Notifications.ToastNotificationManager"
	winrtSlotXmlDocumentCreateElem = com.IInspectableSlots + 3
	winrtSlotXmlNodeAppendChild    = com.IInspectableSlots + 16
	winrtSlotXmlElementSetAttr     = com.IInspectableSlots + 2
	winrtSlotXmlSetInnerText       = com.IInspectableSlots + 2
	winrtSlotToastActivated        = com.IInspectableSlots + 5
	winrtSlotToastSetTag           = com.IInspectableSlots + 0
	winrtSlotToastFactoryCreate    = com.IInspectableSlots + 0
	winrtSlotManagerCreateNotifier = com.IInspectableSlots + 0
	winrtSlotManagerCreateWithID   = com.IInspectableSlots + 1
	winrtSlotNotifierShow          = com.IInspectableSlots + 0
	winrtSlotNotifierHide          = com.IInspectableSlots + 1
	winrtSlotActivatedArguments    = com.IInspectableSlots + 0
)

type toastNotifier struct{ com.IInspectable }

func (n *toastNotifier) show(toast *toastNotification) error {
	return com.CallHR(n.Ptr(), winrtSlotNotifierShow, uintptr(toast.Ptr()))
}

func (n *toastNotifier) hide(toast *toastNotification) error {
	return com.CallHR(n.Ptr(), winrtSlotNotifierHide, uintptr(toast.Ptr()))
}

type toastNotification struct{ com.IInspectable }

type xmlDocument struct{ com.IInspectable }

type xmlElement struct{ com.IInspectable }

func withHSTRING(s string, f func(com.HSTRING) error) error {
	h, err := com.NewHSTRING(s)
	if err != nil {
		return err
	}
	defer h.Delete()
	return f(h)
}

type responseCallback struct {
	fn func(gpui.SystemNotificationResponse)
}

// SystemNotificationState shows system notifications as Windows toast notifications.
type SystemNotificationState struct {
	notifier         *toastNotifier
	activeToasts     map[gpui.SharedString]*toastNotification
	responseSender   *scheduler.ChanSender[gpui.SystemNotificationResponse]
	responseReceiver *scheduler.ChanReceiver[gpui.SystemNotificationResponse]
	callback         *responseCallback
	responseTask     *gpui.Task[struct{}]
}

type appIdentity struct {
	identifier string
	name       string
}

// NewSystemNotificationState returns state with no notifier yet.
func NewSystemNotificationState() *SystemNotificationState {
	sender, receiver := scheduler.Chan[gpui.SystemNotificationResponse](0)
	return &SystemNotificationState{
		activeToasts:     map[gpui.SharedString]*toastNotification{},
		responseSender:   sender,
		responseReceiver: receiver,
		callback:         &responseCallback{},
	}
}

func (s *SystemNotificationState) show(hasPackageIdentity bool, identity gpui.Option[appIdentity], notification gpui.SystemNotification) error {
	notifier, ok, err := s.getNotifier(hasPackageIdentity, identity)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	document, err := toastDocument(notification)
	if err != nil {
		return err
	}
	defer document.Release()

	factory, err := com.RoGetActivationFactory(toastNotificationClass, &iidIToastNotificationFactory)
	if err != nil {
		return err
	}
	var toast *toastNotification
	err = com.CallHR(factory.Ptr(), winrtSlotToastFactoryCreate, uintptr(document.Ptr()), uintptr(unsafe.Pointer(&toast)))
	factory.Release()
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			toast.Release()
		}
	}()

	tag := fmt.Sprintf("%016x", hashString(string(notification.Tag)))
	toast2, err := com.Cast[com.IInspectable](&toast.IUnknown, &iidIToastNotification2)
	if err != nil {
		return err
	}
	err = withHSTRING(tag, func(h com.HSTRING) error {
		return com.CallHR(toast2.Ptr(), winrtSlotToastSetTag, uintptr(h))
	})
	toast2.Release()
	if err != nil {
		return err
	}

	handler := newToastActivatedHandler(s.responseSender, notification.Tag)
	var token int64
	err = com.CallHR(toast.Ptr(), winrtSlotToastActivated, uintptr(unsafe.Pointer(handler.Interface(0))), uintptr(unsafe.Pointer(&token)))
	handler.Release()
	if err != nil {
		return err
	}

	if previous, ok := s.activeToasts[notification.Tag]; ok {
		delete(s.activeToasts, notification.Tag)
		err := notifier.hide(previous)
		previous.Release()
		if err != nil {
			return err
		}
	}
	if err := notifier.show(toast); err != nil {
		return err
	}
	keep = true
	s.activeToasts[notification.Tag] = toast
	return nil
}

func (s *SystemNotificationState) dismiss(tag string) {
	toast, ok := s.activeToasts[gpui.SharedString(tag)]
	if !ok {
		return
	}
	delete(s.activeToasts, gpui.SharedString(tag))
	defer toast.Release()
	if s.notifier == nil {
		return
	}
	if err := s.notifier.hide(toast); err != nil {
		slog.Warn(fmt.Sprintf("failed to dismiss system notification: %v", err))
	}
}

func (s *SystemNotificationState) onResponse(executor gpui.ForegroundExecutor, callback func(gpui.SystemNotificationResponse)) {
	s.callback.fn = callback

	if receiver := s.responseReceiver; receiver != nil {
		s.responseReceiver = nil
		cb := s.callback
		s.responseTask = executor.Spawn(func(sc *scheduler.Scope) (struct{}, error) {
			defer receiver.Close()
			for {
				response, ok := receiver.Recv(sc)
				if !ok {
					return struct{}{}, nil
				}
				taken := cb.fn
				cb.fn = nil
				if taken != nil {
					taken(response)
					if cb.fn == nil {
						cb.fn = taken
					}
				}
			}
		})
	}
}

func (s *SystemNotificationState) getNotifier(hasPackageIdentity bool, identity gpui.Option[appIdentity]) (*toastNotifier, bool, error) {
	if s.notifier != nil {
		return s.notifier, true, nil
	}

	statics, err := com.RoGetActivationFactory(toastNotificationManagerClass, &iidIToastNotificationManagerStatics)
	if err != nil {
		return nil, false, err
	}
	defer statics.Release()

	var notifier *toastNotifier
	if hasPackageIdentity {
		if err := com.CallHR(statics.Ptr(), winrtSlotManagerCreateNotifier, uintptr(unsafe.Pointer(&notifier))); err != nil {
			return nil, false, err
		}
	} else {
		id, ok := identity.Get()
		if !ok {
			slog.Warn("cannot show a system notification without an app identity; " +
				"call `App::set_app_identity` during startup")
			return nil, false, nil
		}
		registerAppUserModelID(id.identifier, id.name)
		err := withHSTRING(id.identifier, func(h com.HSTRING) error {
			return com.CallHR(statics.Ptr(), winrtSlotManagerCreateWithID, uintptr(h), uintptr(unsafe.Pointer(&notifier)))
		})
		if err != nil {
			return nil, false, err
		}
	}

	s.notifier = notifier
	return notifier, true, nil
}

func createXMLElement(document *xmlDocument, name string) (*xmlElement, error) {
	var element *xmlElement
	err := withHSTRING(name, func(h com.HSTRING) error {
		return com.CallHR(document.Ptr(), winrtSlotXmlDocumentCreateElem, uintptr(h), uintptr(unsafe.Pointer(&element)))
	})
	return element, err
}

func appendXMLChild(parent, child *com.IUnknown) error {
	parentNode, err := com.Cast[com.IInspectable](parent, &iidIXmlNode)
	if err != nil {
		return err
	}
	defer parentNode.Release()
	childNode, err := com.Cast[com.IInspectable](child, &iidIXmlNode)
	if err != nil {
		return err
	}
	defer childNode.Release()
	var appended *com.IInspectable
	if err := com.CallHR(parentNode.Ptr(), winrtSlotXmlNodeAppendChild, uintptr(childNode.Ptr()), uintptr(unsafe.Pointer(&appended))); err != nil {
		return err
	}
	if appended != nil {
		appended.Release()
	}
	return nil
}

func setXMLAttribute(element *xmlElement, name, value string) error {
	return withHSTRING(name, func(n com.HSTRING) error {
		return withHSTRING(value, func(v com.HSTRING) error {
			return com.CallHR(element.Ptr(), winrtSlotXmlElementSetAttr, uintptr(n), uintptr(v))
		})
	})
}

func setXMLInnerText(element *xmlElement, text string) error {
	serializer, err := com.Cast[com.IInspectable](&element.IUnknown, &iidIXmlNodeSerializer)
	if err != nil {
		return err
	}
	defer serializer.Release()
	return withHSTRING(text, func(h com.HSTRING) error {
		return com.CallHR(serializer.Ptr(), winrtSlotXmlSetInnerText, uintptr(h))
	})
}

func toastDocument(notification gpui.SystemNotification) (document *xmlDocument, err error) {
	instance, err := com.RoActivateInstance(xmlDocumentClass)
	if err != nil {
		return nil, err
	}
	document, err = com.Cast[xmlDocument](&instance.IUnknown, &iidIXmlDocument)
	instance.Release()
	if err != nil {
		return nil, err
	}
	var elements []*xmlElement
	defer func() {
		for _, e := range elements {
			e.Release()
		}
		if err != nil {
			document.Release()
			document = nil
		}
	}()
	create := func(name string) (*xmlElement, error) {
		e, err := createXMLElement(document, name)
		if err != nil {
			return nil, err
		}
		elements = append(elements, e)
		return e, nil
	}

	toast, err := create("toast")
	if err != nil {
		return nil, err
	}
	if err := appendXMLChild(&document.IUnknown, &toast.IUnknown); err != nil {
		return nil, err
	}

	visual, err := create("visual")
	if err != nil {
		return nil, err
	}
	if err := appendXMLChild(&toast.IUnknown, &visual.IUnknown); err != nil {
		return nil, err
	}
	binding, err := create("binding")
	if err != nil {
		return nil, err
	}
	if err := setXMLAttribute(binding, "template", "ToastGeneric"); err != nil {
		return nil, err
	}
	if err := appendXMLChild(&visual.IUnknown, &binding.IUnknown); err != nil {
		return nil, err
	}
	for _, text := range []gpui.SharedString{notification.Title, notification.Body} {
		element, err := create("text")
		if err != nil {
			return nil, err
		}
		if err := setXMLInnerText(element, string(text)); err != nil {
			return nil, err
		}
		if err := appendXMLChild(&binding.IUnknown, &element.IUnknown); err != nil {
			return nil, err
		}
	}

	if len(notification.Actions) != 0 {
		actions, err := create("actions")
		if err != nil {
			return nil, err
		}
		if err := appendXMLChild(&toast.IUnknown, &actions.IUnknown); err != nil {
			return nil, err
		}
		for _, action := range notification.Actions {
			actionElement, err := create("action")
			if err != nil {
				return nil, err
			}
			if err := setXMLAttribute(actionElement, "content", string(action.Label)); err != nil {
				return nil, err
			}
			if err := setXMLAttribute(actionElement, "arguments", string(action.ID)); err != nil {
				return nil, err
			}
			if err := appendXMLChild(&actions.IUnknown, &actionElement.IUnknown); err != nil {
				return nil, err
			}
		}
	}

	audio, err := create("audio")
	if err != nil {
		return nil, err
	}
	if err := setXMLAttribute(audio, "silent", "true"); err != nil {
		return nil, err
	}
	if err := appendXMLChild(&toast.IUnknown, &audio.IUnknown); err != nil {
		return nil, err
	}
	return document, nil
}

func registerAppUserModelID(appIdentifier, appName string) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\AppUserModelId\`+appIdentifier, registry.SET_VALUE)
	if err == nil {
		err = key.SetStringValue("DisplayName", appName)
		key.Close()
	}
	if err != nil {
		slog.Warn(fmt.Sprintf("failed to register AppUserModelID; notifications may not display: %v", err))
	}
}

type toastActivatedHandler struct {
	sender *scheduler.ChanSender[gpui.SystemNotificationResponse]
	tag    gpui.SharedString
}

func newToastActivatedHandler(sender *scheduler.ChanSender[gpui.SystemNotificationResponse], tag gpui.SharedString) *com.Object {
	return com.NewObject(&toastActivatedHandler{sender: sender, tag: tag}, toastActivatedHandlerVtable.Get())
}

func (h *toastActivatedHandler) invoke(arguments *com.IInspectable) com.HRESULT {
	actionID := gpui.None[gpui.SharedString]()
	if arguments != nil {
		if args, err := com.Cast[com.IInspectable](&arguments.IUnknown, &iidIToastActivatedEventArgs); err == nil {
			var value com.HSTRING
			if com.CallHR(args.Ptr(), winrtSlotActivatedArguments, uintptr(unsafe.Pointer(&value))) == nil {
				if s := value.String(); s != "" {
					actionID = gpui.Some(gpui.SharedString(s))
				}
				value.Delete()
			}
			args.Release()
		}
	}
	_ = h.sender.TrySend(gpui.SystemNotificationResponse{Tag: h.tag, ActionID: actionID})
	return com.S_OK
}

var toastActivatedHandlerVtable = com.NewLazyVtable(func() *com.Vtable {
	return com.NewVtable([]com.GUID{iidToastActivatedHandler, com.IID_IAgileObject},
		xwin.NewCallback(func(this *com.IUnknown, sender *com.IInspectable, arguments *com.IInspectable) uintptr {
			impl, ok := com.ImplOf[*toastActivatedHandler](this)
			if !ok {
				return com.E_FAIL.Uintptr()
			}
			return impl.invoke(arguments).Uintptr()
		}),
	)
})

func hashString(s string) uint64 {
	h := newSipHasher13()
	h.write([]byte(s))
	h.write([]byte{0xff})
	return h.finish()
}

type sipHasher13 struct {
	v0, v1, v2, v3 uint64
	tail           uint64
	ntail          int
	length         int
}

func newSipHasher13() *sipHasher13 {
	return &sipHasher13{
		v0: 0x736f6d6570736575,
		v1: 0x646f72616e646f6d,
		v2: 0x6c7967656e657261,
		v3: 0x7465646279746573,
	}
}

func (h *sipHasher13) round() {
	h.v0 += h.v1
	h.v1 = bits.RotateLeft64(h.v1, 13)
	h.v1 ^= h.v0
	h.v0 = bits.RotateLeft64(h.v0, 32)
	h.v2 += h.v3
	h.v3 = bits.RotateLeft64(h.v3, 16)
	h.v3 ^= h.v2
	h.v0 += h.v3
	h.v3 = bits.RotateLeft64(h.v3, 21)
	h.v3 ^= h.v0
	h.v2 += h.v1
	h.v1 = bits.RotateLeft64(h.v1, 17)
	h.v1 ^= h.v2
	h.v2 = bits.RotateLeft64(h.v2, 32)
}

func (h *sipHasher13) compress(m uint64) {
	h.v3 ^= m
	h.round()
	h.v0 ^= m
}

func (h *sipHasher13) write(msg []byte) {
	h.length += len(msg)
	for _, b := range msg {
		h.tail |= uint64(b) << (8 * h.ntail)
		h.ntail++
		if h.ntail == 8 {
			h.compress(h.tail)
			h.tail = 0
			h.ntail = 0
		}
	}
}

func (h *sipHasher13) finish() uint64 {
	b := (uint64(h.length)&0xff)<<56 | h.tail
	h.compress(b)
	h.v2 ^= 0xff
	h.round()
	h.round()
	h.round()
	return h.v0 ^ h.v1 ^ h.v2 ^ h.v3
}
