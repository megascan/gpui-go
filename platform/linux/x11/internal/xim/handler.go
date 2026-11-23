//go:build linux

package xim

// Handler receives the decoded replies and notifications of an XIM server.
// Each method corresponds to one ClientHandler callback of zed-xim.
type Handler interface {
	HandleConnect(c *Client) error
	HandleDisconnect()
	HandleOpen(c *Client, imID uint16) error
	HandleClose(c *Client, imID uint16) error
	HandleQueryExtension(c *Client, extensions []Extension) error
	HandleGetIMValues(c *Client, imID uint16, values map[AttributeName][]byte) error
	HandleSetICValues(c *Client, imID, icID uint16) error
	HandleCreateIC(c *Client, imID, icID uint16) error
	HandleDestroyIC(c *Client, imID, icID uint16) error
	HandleCommit(c *Client, imID, icID uint16, text string) error
	HandleForwardEvent(c *Client, imID, icID uint16, flag ForwardEventFlag, event XEvent) error
	HandleSetEventMask(c *Client, imID, icID uint16, forwardMask, syncMask uint32) error
	HandlePreeditStart(c *Client, imID, icID uint16) error
	HandlePreeditDraw(c *Client, imID, icID uint16, caret, chgFirst, chgLength int32, status PreeditDrawStatus, text string, feedbacks []Feedback) error
	HandlePreeditCaret(c *Client, imID, icID uint16, position *int32, direction CaretDirection, style CaretStyle) error
	HandlePreeditDone(c *Client, imID, icID uint16) error
	HandleResetIC(c *Client, imID, icID uint16, preeditText string) error
}

// NopHandler implements Handler with no-op callbacks; embed it and override what is needed.
type NopHandler struct{}

func (NopHandler) HandleConnect(*Client) error { return nil }

func (NopHandler) HandleDisconnect() {}

func (NopHandler) HandleOpen(*Client, uint16) error { return nil }

func (NopHandler) HandleClose(*Client, uint16) error { return nil }

func (NopHandler) HandleQueryExtension(*Client, []Extension) error { return nil }

func (NopHandler) HandleGetIMValues(*Client, uint16, map[AttributeName][]byte) error { return nil }

func (NopHandler) HandleSetICValues(*Client, uint16, uint16) error { return nil }

func (NopHandler) HandleCreateIC(*Client, uint16, uint16) error { return nil }

func (NopHandler) HandleDestroyIC(*Client, uint16, uint16) error { return nil }

func (NopHandler) HandleCommit(*Client, uint16, uint16, string) error { return nil }

func (NopHandler) HandleForwardEvent(*Client, uint16, uint16, ForwardEventFlag, XEvent) error {
	return nil
}

func (NopHandler) HandleSetEventMask(*Client, uint16, uint16, uint32, uint32) error { return nil }

func (NopHandler) HandlePreeditStart(*Client, uint16, uint16) error { return nil }

func (NopHandler) HandlePreeditDraw(*Client, uint16, uint16, int32, int32, int32, PreeditDrawStatus, string, []Feedback) error {
	return nil
}

func (NopHandler) HandlePreeditCaret(*Client, uint16, uint16, *int32, CaretDirection, CaretStyle) error {
	return nil
}

func (NopHandler) HandlePreeditDone(*Client, uint16, uint16) error { return nil }

func (NopHandler) HandleResetIC(*Client, uint16, uint16, string) error { return nil }
