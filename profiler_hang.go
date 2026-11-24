package gpui

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"slices"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

// MeasurementVersion is the version of the power/visibility-aware measurement rules.
const MeasurementVersion uint32 = 3

// HangDetector detects foreground hangs by polling the journal. Detection is post-hoc: a hang
// is reported once an explicit presentation, completed frame skip, or foreground-idle boundary
// completes its interval.
type HangDetector struct {
	collector      *ForegroundJournalCollector
	sealer         *IntervalSealer
	threshold      time.Duration
	frameBudget    time.Duration
	firstPresentAt Option[scheduler.Instant]
	activity       activeUse
}

// ActiveUseWindow is how long the user counts as actively using the app after deliberate input.
const ActiveUseWindow = 60 * time.Second

var passiveInputKinds = [3]string{"mouse_move", "mouse_exited", "mouse_pressure"}

type activeUse struct {
	recentInputs             []scheduler.Instant
	accountedUntil           Option[scheduler.Instant]
	activeUntil              Option[scheduler.Instant]
	firstInputSinceAccounted Option[scheduler.Instant]
	lastInputSinceAccounted  Option[scheduler.Instant]
	suspended                bool
}

func (a *activeUse) observe(entry ForegroundJournalEntry) {
	switch entry := entry.(type) {
	case ForegroundJournalEntryEvent:
		input, ok := entry.Event.(ForegroundEventInput)
		if !ok || slices.Contains(passiveInputKinds[:], input.Kind) {
			return
		}
		a.recentInputs = append(a.recentInputs, input.End)
		if a.firstInputSinceAccounted.IsNone() {
			a.firstInputSinceAccounted = Some(input.Start)
		}
		if last, ok := a.lastInputSinceAccounted.Get(); ok {
			a.lastInputSinceAccounted = Some(instantMax(last, input.End))
		} else {
			a.lastInputSinceAccounted = Some(input.End)
		}
	case ForegroundJournalEntryBoundary:
		if _, ok := entry.Boundary.(IntervalBoundaryPowerTransition); ok {
			a.suspended = true
		}
	}
}

func (a *activeUse) wasActiveDuring(start, end scheduler.Instant) bool {
	since := start.Add(-ActiveUseWindow)
	return slices.ContainsFunc(a.recentInputs, func(input scheduler.Instant) bool {
		return !input.Before(since) && !input.After(end)
	})
}

func (a *activeUse) take(now scheduler.Instant, maxElapsed time.Duration) time.Duration {
	retainedSince := now.Add(-ActiveUseWindow * 10)
	for len(a.recentInputs) > 0 && a.recentInputs[0].Before(retainedSince) {
		a.recentInputs = a.recentInputs[1:]
	}
	since := a.accountedUntil.Replace(now)
	previousUntil := a.activeUntil
	firstInput := a.firstInputSinceAccounted.Take()
	if lastInput, ok := a.lastInputSinceAccounted.Take().Get(); ok {
		until := lastInput.Add(ActiveUseWindow)
		if previous, ok := previousUntil.Get(); ok {
			a.activeUntil = Some(instantMax(previous, until))
		} else {
			a.activeUntil = Some(until)
		}
	}
	suspended := a.suspended
	a.suspended = false
	sinceValue, ok := since.Get()
	if !ok || suspended {
		return 0
	}
	continuedUntil := sinceValue
	if until, ok := previousUntil.Get(); ok {
		continuedUntil = instantMax(until.Min(now), sinceValue)
	}
	active := continuedUntil.SaturatingDurationSince(sinceValue)
	if first, ok := firstInput.Get(); ok {
		if activeUntil, ok := a.activeUntil.Get(); ok {
			start := instantMax(first, continuedUntil)
			active += activeUntil.Min(now).SaturatingDurationSince(start)
		}
	}
	return min(active, maxElapsed)
}

// HangIncident is one sealed interval that contained at least one hang.
type HangIncident struct {
	// Snapshot is the interval the hangs occurred in, including all non-hang foreground work
	// recorded alongside them.
	Snapshot FrameSnapshot
	// Trigger is the detection rule that qualified the interval.
	Trigger HangTrigger
	// Contributors are the events that caused the incident, longest first.
	Contributors []ForegroundEvent
	// DuringActiveUse is whether the user was actively using the app during the incident.
	DuringActiveUse bool
}

// HangTrigger is the detection rule that qualified an interval as a HangIncident.
type HangTrigger uint8

const (
	// HangTriggerThreshold means a single event blocked the foreground for at least the hang
	// threshold.
	HangTriggerThreshold HangTrigger = iota
	// HangTriggerLateFrame means the interval presented a frame whose work exceeded its budget.
	HangTriggerLateFrame
	// HangTriggerBlockingWork means no presented frame's work could be measured, but a single
	// event blocked the foreground for at least the frame budget.
	HangTriggerBlockingWork
)

// HangTriggerAll is every trigger.
var HangTriggerAll = [3]HangTrigger{HangTriggerThreshold, HangTriggerLateFrame, HangTriggerBlockingWork}

// AsStr returns the trigger's serialized name, e.g. "late_frame".
func (t HangTrigger) AsStr() string {
	switch t {
	case HangTriggerThreshold:
		return "threshold"
	case HangTriggerLateFrame:
		return "late_frame"
	default:
		return "blocking_work"
	}
}

// String returns the trigger's variant name.
func (t HangTrigger) String() string {
	switch t {
	case HangTriggerThreshold:
		return "Threshold"
	case HangTriggerLateFrame:
		return "LateFrame"
	default:
		return "BlockingWork"
	}
}

// MarshalText encodes the trigger as its serialized name.
func (t HangTrigger) MarshalText() ([]byte, error) {
	return []byte(t.AsStr()), nil
}

// NewHangDetector creates a detector reporting single events at or above threshold, late
// frames, and blocking work of at least frameBudget. Only events recorded from this point on
// are observed.
func NewHangDetector(journal ForegroundJournal, threshold time.Duration, frameBudget time.Duration) *HangDetector {
	return &HangDetector{
		collector:   journal.Collector(),
		sealer:      NewIntervalSealer(scheduler.InstantNow()),
		threshold:   threshold,
		frameBudget: frameBudget,
	}
}

// TakeActiveTime returns the time since the previous call that the user was actively using the
// app, as of the journal entries polled so far, capped at maxElapsed.
func (d *HangDetector) TakeActiveTime(now scheduler.Instant, maxElapsed time.Duration) time.Duration {
	return d.activity.take(now, maxElapsed)
}

// FirstPresentAt returns when the first newly drawn frame observed by this detector finished
// platform submission.
func (d *HangDetector) FirstPresentAt() (scheduler.Instant, bool) {
	return d.firstPresentAt.Get()
}

// Poll drains newly recorded events and returns the incidents sealed since the previous poll.
func (d *HangDetector) Poll() []HangIncident {
	drained := d.collector.CollectUnseen()
	for _, entry := range drained.Entries {
		d.activity.observe(entry)
	}
	if d.firstPresentAt.IsNone() {
		for _, entry := range drained.Entries {
			if boundary, ok := entry.(ForegroundJournalEntryBoundary); ok {
				if presented, ok := boundary.Boundary.(IntervalBoundaryPresented); ok {
					d.firstPresentAt = Some(presented.Presentation.PresentEnd)
					break
				}
			}
		}
	}
	var incidents []HangIncident
	for _, snapshot := range d.sealer.PushEntries(drained.Entries) {
		incident, ok := HangIncidentDetect(snapshot, d.threshold, d.frameBudget)
		if !ok {
			continue
		}
		start, end := incident.ActiveWindow()
		incident.DuringActiveUse = d.activity.wasActiveDuring(start, end)
		incidents = append(incidents, incident)
	}
	return incidents
}

// SerializedHangIncident is a HangIncident in a telemetry-friendly form: timestamps and
// durations in fractional milliseconds since app startup (microsecond precision).
type SerializedHangIncident struct {
	MeasurementVersion   uint32                      `json:"measurement_version"`
	Phase                string                      `json:"phase"`
	Trigger              HangTrigger                 `json:"trigger"`
	DuringActiveUse      bool                        `json:"during_active_use"`
	StartMs              float64                     `json:"start_ms"`
	ActiveMs             float64                     `json:"active_ms"`
	StallMs              float64                     `json:"stall_ms"`
	DirtyToPresentMs     Option[float64]             `json:"dirty_to_present_ms"`
	SealedBy             string                      `json:"sealed_by"`
	BusyFraction         float64                     `json:"busy_fraction"`
	EventCount           int                         `json:"event_count"`
	SmallPollCount       uint64                      `json:"small_poll_count"`
	SmallPollTotalMs     float64                     `json:"small_poll_total_ms"`
	DroppedEvents        uint64                      `json:"dropped_events"`
	JournalDiscontinuous bool                        `json:"journal_discontinuous"`
	Contributors         []SerializedHangContributor `json:"contributors"`
	ContributorsElided   int                         `json:"contributors_elided"`
}

// SerializedHangContributor is one hang contributor in serialized form:
// SerializedHangContributorTaskPoll, SerializedHangContributorAction,
// SerializedHangContributorInput, SerializedHangContributorDraw, SerializedHangContributorView
// or SerializedHangContributorPresent.
type SerializedHangContributor interface {
	isSerializedHangContributor()
}

// SerializedHangContributorTaskPoll is a foreground task poll.
type SerializedHangContributorTaskPoll struct {
	Location   SerializedLocation
	StartMs    float64
	DurationMs float64
	Depth      int
}

// SerializedHangContributorAction is an action handler.
type SerializedHangContributorAction struct {
	Name       string
	StartMs    float64
	DurationMs float64
	Depth      int
}

// SerializedHangContributorInput is a platform input dispatch.
type SerializedHangContributorInput struct {
	InputKind          string
	StartMs            float64
	DurationMs         float64
	CausedInvalidation bool
	Depth              int
}

// SerializedHangContributorDraw is a window draw.
type SerializedHangContributorDraw struct {
	WindowID        uint64
	StartMs         float64
	DurationMs      float64
	DirtyToDrawMs   Option[float64]
	Invalidations   uint64
	RequestLayoutMs float64
	PrepaintMs      float64
	PaintMs         float64
	Depth           int
}

// SerializedHangContributorView is one phase of drawing a view whose own work took at least
// ViewTimingFloor.
type SerializedHangContributorView struct {
	Name       string
	Phase      string
	StartMs    float64
	DurationMs float64
	SelfMs     float64
	Depth      int
}

// SerializedHangContributorPresent is work spent submitting a frame to the platform.
type SerializedHangContributorPresent struct {
	WindowID   uint64
	StartMs    float64
	DurationMs float64
	Depth      int
}

func (SerializedHangContributorTaskPoll) isSerializedHangContributor() {}
func (SerializedHangContributorAction) isSerializedHangContributor()   {}
func (SerializedHangContributorInput) isSerializedHangContributor()    {}
func (SerializedHangContributorDraw) isSerializedHangContributor()     {}
func (SerializedHangContributorView) isSerializedHangContributor()     {}
func (SerializedHangContributorPresent) isSerializedHangContributor()  {}

// MarshalJSONTo encodes the contributor with its "kind" tag.
func (c SerializedHangContributorTaskPoll) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, struct {
		Kind       string             `json:"kind"`
		Location   SerializedLocation `json:"location"`
		StartMs    float64            `json:"start_ms"`
		DurationMs float64            `json:"duration_ms"`
		Depth      int                `json:"depth"`
	}{"task_poll", c.Location, c.StartMs, c.DurationMs, c.Depth})
}

// MarshalJSONTo encodes the contributor with its "kind" tag.
func (c SerializedHangContributorAction) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, struct {
		Kind       string  `json:"kind"`
		Name       string  `json:"name"`
		StartMs    float64 `json:"start_ms"`
		DurationMs float64 `json:"duration_ms"`
		Depth      int     `json:"depth"`
	}{"action", c.Name, c.StartMs, c.DurationMs, c.Depth})
}

// MarshalJSONTo encodes the contributor with its "kind" tag.
func (c SerializedHangContributorInput) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, struct {
		Kind               string  `json:"kind"`
		InputKind          string  `json:"input_kind"`
		StartMs            float64 `json:"start_ms"`
		DurationMs         float64 `json:"duration_ms"`
		CausedInvalidation bool    `json:"caused_invalidation"`
		Depth              int     `json:"depth"`
	}{"input", c.InputKind, c.StartMs, c.DurationMs, c.CausedInvalidation, c.Depth})
}

// MarshalJSONTo encodes the contributor with its "kind" tag.
func (c SerializedHangContributorDraw) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, struct {
		Kind            string          `json:"kind"`
		WindowID        uint64          `json:"window_id"`
		StartMs         float64         `json:"start_ms"`
		DurationMs      float64         `json:"duration_ms"`
		DirtyToDrawMs   Option[float64] `json:"dirty_to_draw_ms"`
		Invalidations   uint64          `json:"invalidations"`
		RequestLayoutMs float64         `json:"request_layout_ms"`
		PrepaintMs      float64         `json:"prepaint_ms"`
		PaintMs         float64         `json:"paint_ms"`
		Depth           int             `json:"depth"`
	}{"draw", c.WindowID, c.StartMs, c.DurationMs, c.DirtyToDrawMs, c.Invalidations, c.RequestLayoutMs, c.PrepaintMs, c.PaintMs, c.Depth})
}

// MarshalJSONTo encodes the contributor with its "kind" tag.
func (c SerializedHangContributorView) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, struct {
		Kind       string  `json:"kind"`
		Name       string  `json:"name"`
		Phase      string  `json:"phase"`
		StartMs    float64 `json:"start_ms"`
		DurationMs float64 `json:"duration_ms"`
		SelfMs     float64 `json:"self_ms"`
		Depth      int     `json:"depth"`
	}{"view", c.Name, c.Phase, c.StartMs, c.DurationMs, c.SelfMs, c.Depth})
}

// MarshalJSONTo encodes the contributor with its "kind" tag.
func (c SerializedHangContributorPresent) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, struct {
		Kind       string  `json:"kind"`
		WindowID   uint64  `json:"window_id"`
		StartMs    float64 `json:"start_ms"`
		DurationMs float64 `json:"duration_ms"`
		Depth      int     `json:"depth"`
	}{"present", c.WindowID, c.StartMs, c.DurationMs, c.Depth})
}

func hangAsMillis(duration time.Duration) float64 {
	return float64(uint64(duration/time.Microsecond)) / 1000.0
}

// SerializedHangIncidentConvert converts an incident, keeping at most maxContributors
// contributors. Incidents whose active window begins before firstPresentAt are tagged
// "startup".
func SerializedHangIncidentConvert(startup scheduler.Instant, incident *HangIncident, maxContributors int, firstPresentAt Option[scheduler.Instant]) SerializedHangIncident {
	sinceStartup := func(instant scheduler.Instant) float64 {
		return hangAsMillis(instant.SaturatingDurationSince(startup))
	}
	snapshot := &incident.Snapshot
	activeStart, activeEnd := incident.ActiveWindow()
	active := activeEnd.SaturatingDurationSince(activeStart)
	busyFraction := 1.0
	if active != 0 {
		busyFraction = min(durationDivDurationF64(snapshot.OccupancyWithin(activeStart, activeEnd), active), 1.0)
	}
	phase := "startup"
	if fp, ok := firstPresentAt.Get(); ok && !activeStart.Before(fp) {
		phase = "steady"
	}
	stallMs := 0.0
	if len(incident.Contributors) > 0 {
		stallMs = hangAsMillis(incident.Contributors[0].Duration())
	}
	dirtyToPresentMs := None[float64]()
	var sealedBy string
	switch boundary := snapshot.Boundary.(type) {
	case IntervalBoundaryPresented:
		if d, ok := boundary.DirtyToPresentDuration(); ok {
			dirtyToPresentMs = Some(hangAsMillis(d))
		}
		sealedBy = "present"
	case IntervalBoundaryFrameSkipped:
		sealedBy = "frame_skipped"
	case IntervalBoundaryIdle:
		sealedBy = "idle"
	case IntervalBoundaryPowerTransition:
		sealedBy = "power_transition"
	}
	kept := slices.Clone(incident.Contributors[:min(maxContributors, len(incident.Contributors))])
	slices.SortStableFunc(kept, func(a, b ForegroundEvent) int { return a.StartTime().Compare(b.StartTime()) })
	contributors := make([]SerializedHangContributor, 0, len(kept))
	for _, event := range kept {
		contributors = append(contributors, serializedHangContributorConvert(startup, event, hangNestingDepth(event, snapshot.Events)))
	}
	return SerializedHangIncident{
		Phase:                phase,
		MeasurementVersion:   MeasurementVersion,
		Trigger:              incident.Trigger,
		DuringActiveUse:      incident.DuringActiveUse,
		StartMs:              sinceStartup(activeStart),
		ActiveMs:             hangAsMillis(active),
		StallMs:              stallMs,
		DirtyToPresentMs:     dirtyToPresentMs,
		SealedBy:             sealedBy,
		BusyFraction:         math.Round(busyFraction*1000.0) / 1000.0,
		EventCount:           len(snapshot.Events),
		SmallPollCount:       snapshot.SmallPollSummary().Count,
		SmallPollTotalMs:     hangAsMillis(snapshot.SmallPollSummary().Total),
		DroppedEvents:        snapshot.DroppedEvents,
		JournalDiscontinuous: snapshot.JournalDiscontinuous,
		Contributors:         contributors,
		ContributorsElided:   max(len(incident.Contributors)-maxContributors, 0),
	}
}

func hangNestingDepth(event ForegroundEvent, events []ForegroundEvent) int {
	start, end := event.StartTime(), event.EndTime()
	count := 0
	for _, other := range events {
		otherStart, otherEnd := other.StartTime(), other.EndTime()
		if !otherStart.After(start) && !end.After(otherEnd) && (otherStart.Before(start) || end.Before(otherEnd)) {
			count++
		}
	}
	return count
}

func serializedHangContributorConvert(startup scheduler.Instant, event ForegroundEvent, depth int) SerializedHangContributor {
	sinceStartup := func(instant scheduler.Instant) float64 {
		return hangAsMillis(instant.SaturatingDurationSince(startup))
	}
	durationMs := hangAsMillis(event.Duration())
	switch timing := event.(type) {
	case ForegroundEventTaskPoll:
		return SerializedHangContributorTaskPoll{
			Location:   serializedLocationFrom(timing.Location),
			StartMs:    sinceStartup(timing.Start),
			DurationMs: durationMs,
			Depth:      depth,
		}
	case ForegroundEventAction:
		return SerializedHangContributorAction{
			Name:       timing.Name,
			StartMs:    sinceStartup(timing.Start),
			DurationMs: durationMs,
			Depth:      depth,
		}
	case ForegroundEventInput:
		return SerializedHangContributorInput{
			InputKind:          timing.Kind,
			StartMs:            sinceStartup(timing.Start),
			DurationMs:         durationMs,
			CausedInvalidation: timing.CausedInvalidation,
			Depth:              depth,
		}
	case ForegroundEventDraw:
		dirtyToDraw := None[float64]()
		if d, ok := timing.DirtyToDrawDuration(); ok {
			dirtyToDraw = Some(hangAsMillis(d))
		}
		return SerializedHangContributorDraw{
			WindowID:        timing.WindowID.AsU64(),
			StartMs:         sinceStartup(timing.DrawStart),
			DurationMs:      durationMs,
			DirtyToDrawMs:   dirtyToDraw,
			Invalidations:   timing.Invalidations,
			RequestLayoutMs: hangAsMillis(timing.Phases.RequestLayout),
			PrepaintMs:      hangAsMillis(timing.Phases.Prepaint),
			PaintMs:         hangAsMillis(timing.Phases.Paint),
			Depth:           depth,
		}
	case ForegroundEventView:
		return SerializedHangContributorView{
			Name:       timing.Name,
			Phase:      timing.Phase.AsStr(),
			StartMs:    sinceStartup(timing.Start),
			DurationMs: durationMs,
			SelfMs:     hangAsMillis(timing.SelfDuration),
			Depth:      depth,
		}
	case ForegroundEventPresent:
		return SerializedHangContributorPresent{
			WindowID:   timing.WindowID.AsU64(),
			StartMs:    sinceStartup(timing.PresentStart),
			DurationMs: durationMs,
			Depth:      depth,
		}
	case ForegroundEventSmallPolls:
		return SerializedHangContributorTaskPoll{
			Location: SerializedLocation{
				File:   "<small poll summary>",
				Line:   0,
				Column: 0,
			},
			StartMs:    sinceStartup(timing.Since),
			DurationMs: durationMs,
			Depth:      depth,
		}
	default:
		panic("gpui: unknown foreground event")
	}
}

// ActiveWindow returns the incident's reporting window: from its earliest cause (the sealing
// frame's first invalidation, or the earliest contributor's start when no repaint was pending)
// to the seal.
func (i *HangIncident) ActiveWindow() (scheduler.Instant, scheduler.Instant) {
	snapshot := &i.Snapshot
	start, hasStart := snapshot.Boundary.DirtyAt()
	for _, event := range i.Contributors {
		if eventStart := event.StartTime(); !hasStart || eventStart.Before(start) {
			start, hasStart = eventStart, true
		}
	}
	if !hasStart {
		start = snapshot.IntervalStart
	}
	return start, snapshot.IntervalEnd()
}

// HangIncidentDetect returns an incident when the snapshot contains an event that blocked the
// foreground for threshold or longer; otherwise when the interval presented a late frame,
// unless the journal lost events; otherwise, when no presented frame's work could be measured,
// when an event blocked the foreground for frameBudget or longer.
func HangIncidentDetect(snapshot FrameSnapshot, threshold time.Duration, frameBudget time.Duration) (HangIncident, bool) {
	eventsAtLeast := func(minimum time.Duration) []ForegroundEvent {
		var events []ForegroundEvent
		for _, event := range snapshot.Events {
			if event.Duration() >= minimum {
				events = append(events, event)
			}
		}
		return events
	}
	contributors := eventsAtLeast(threshold)
	var measured *IntervalBoundaryPresented
	var workStart scheduler.Instant
	if presented, ok := snapshot.Boundary.(IntervalBoundaryPresented); ok {
		if start, ok := presented.Frame.WorkStart(); ok {
			measured, workStart = &presented, start
		}
	}
	var trigger HangTrigger
	switch {
	case len(contributors) > 0:
		trigger = HangTriggerThreshold
	case measured != nil:
		submittedAt := measured.Presentation.PresentStart
		late, ok := measured.Frame.IsLate(submittedAt)
		if snapshot.JournalDiscontinuous || !(ok && late) {
			return HangIncident{}, false
		}
		contributors = nil
		for _, event := range snapshot.Events {
			if event.EndTime().After(workStart) && event.StartTime().Before(submittedAt) {
				contributors = append(contributors, event)
			}
		}
		trigger = HangTriggerLateFrame
	default:
		contributors = eventsAtLeast(frameBudget)
		if len(contributors) == 0 {
			return HangIncident{}, false
		}
		trigger = HangTriggerBlockingWork
	}
	slices.SortStableFunc(contributors, func(a, b ForegroundEvent) int { return cmp.Compare(b.Duration(), a.Duration()) })
	return HangIncident{
		Snapshot:        snapshot,
		Trigger:         trigger,
		Contributors:    contributors,
		DuringActiveUse: false,
	}, true
}
