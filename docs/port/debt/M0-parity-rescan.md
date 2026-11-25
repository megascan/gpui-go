# M0-parity-rescan debt

- `scan.areasFromArgs` reads `-area` from `os.Args` because `matrix.Rescanner` has no areas parameter. Fix: add the parameter in matrix/rescan.go and main.go (not owned by this stage), then drop areasFromArgs.
