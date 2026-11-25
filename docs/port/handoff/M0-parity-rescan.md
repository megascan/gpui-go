# M0-parity-rescan handoff

- `internal/cmd/paritycheck/main.go` (owned by M0-bootstrap) must add the blank import `_ "github.com/megascan/gpui-go/internal/cmd/paritycheck/scan"`. The scan package registers `matrix.Rescanner` from `init`; without the import `-rescan` exits 1 with "no scanner registered", so verify line 2 fails. scan imports matrix, so registration cannot move into matrix (cycle).
- Optional: pass the `-area` list to the hook (`Rescanner func(root string, m *Matrix, areas []string, w io.Writer) error`); scan reads `os.Args` today (see docs/port/debt/M0-parity-rescan.md).
