//go:build darwin

package apple

import (
	"log/slog"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/scheduler"
)

const (
	nsProcessInfoThermalStateFair     objc.NSInteger = 1
	nsProcessInfoThermalStateSerious  objc.NSInteger = 2
	nsProcessInfoThermalStateCritical objc.NSInteger = 3
)

// ThermalState returns the system's thermal state.
func ThermalState() gpui.ThermalState {
	switch objc.Send[objc.NSInteger](processInfo(), objc.Sel("thermalState")) {
	case nsProcessInfoThermalStateFair:
		return gpui.ThermalStateFair
	case nsProcessInfoThermalStateSerious:
		return gpui.ThermalStateSerious
	case nsProcessInfoThermalStateCritical:
		return gpui.ThermalStateCritical
	default:
		return gpui.ThermalStateNominal
	}
}

// PowerMode returns the system's power mode. This API does not distinguish Automatic from High
// Power Mode on macOS.
func PowerMode() gpui.PowerMode {
	if objc.Send[bool](processInfo(), objc.Sel("isLowPowerModeEnabled")) {
		return gpui.PowerModeLowPower
	}
	return gpui.PowerModeUnknown
}

// ObserveThermalState calls callback on executor after each thermal state change until the
// observer is closed.
func ObserveThermalState(executor *gpui.ForegroundExecutor, callback func()) *ProcessInfoObserver {
	ThermalState()
	return newProcessInfoObserver(
		objc.FoundationConstant("NSProcessInfoThermalStateDidChangeNotification"),
		executor,
		callback,
	)
}

// ObservePowerMode calls callback on executor after each power state change until the observer is
// closed.
func ObservePowerMode(executor *gpui.ForegroundExecutor, callback func()) *ProcessInfoObserver {
	return newProcessInfoObserver(
		objc.FoundationConstant("NSProcessInfoPowerStateDidChangeNotification"),
		executor,
		callback,
	)
}

// ProcessInfoObserver is a notification-center registration for one NSProcessInfo notification.
// Close is its drop: it removes the observer and cancels the delivery task.
type ProcessInfoObserver struct {
	center   *objc.Retained
	observer *objc.Retained
	task     *gpui.Task[struct{}]
	closed   bool
}

func newProcessInfoObserver(name objc.ID, executor *gpui.ForegroundExecutor, callback func()) *ProcessInfoObserver {
	center := objc.SendClass[objc.ID](objc.GetClass("NSNotificationCenter"), objc.Sel("defaultCenter"))
	return processInfoObserverWithCenter(objc.RetainObject(center), name, executor, callback)
}

func processInfoObserverWithCenter(center *objc.Retained, name objc.ID, executor *gpui.ForegroundExecutor, callback func()) *ProcessInfoObserver {
	sender, receiver := scheduler.Chan[struct{}](0)
	block := objc.NewBlock(func(_ objc.Block, _ objc.ID) {
		if sender.TrySend(struct{}{}) != nil {
			slog.Debug("process info observer stopped before notification delivery")
		}
	})
	observer := objc.Send[objc.ID](
		center.ID(),
		objc.Sel("addObserverForName:object:queue:usingBlock:"),
		name,
		processInfo(),
		objc.Nil,
		block,
	)
	retainedObserver := objc.RetainObject(observer)
	block.Release()
	task := executor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		defer receiver.Close()
		for {
			if _, ok := receiver.Recv(s); !ok {
				break
			}
			callback()
		}
		return struct{}{}, nil
	})
	return &ProcessInfoObserver{
		center:   center,
		observer: retainedObserver,
		task:     task,
	}
}

// Close removes the observer from its notification center and cancels the delivery task. It is
// idempotent.
func (o *ProcessInfoObserver) Close() {
	if o.closed {
		return
	}
	o.closed = true
	objc.SendVoid(o.center.ID(), objc.Sel("removeObserver:"), o.observer.ID())
	o.center.Release()
	o.observer.Release()
	o.task.Cancel()
}
