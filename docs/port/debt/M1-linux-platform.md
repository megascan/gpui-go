# M1-linux-platform debt

- platform/linux/platform.go `fileChooserRequest` duplicates the portal request/Response plumbing of internal/common's unexported `portalCall` (which drops the Response results the FileChooser needs). Export a results-returning variant from common and use it here.
- `fileChooserRequest` waits for the portal Response without a timeout, as ashpd does; the dialog is interactive.
