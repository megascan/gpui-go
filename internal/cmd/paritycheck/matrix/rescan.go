package matrix

import "io"

// Rescanner is the upstream scanner hook. The scan package registers itself here from its init, and the
// paritycheck command imports it, so -rescan reports rows missing from the matrix.
var Rescanner func(root string, m *Matrix, w io.Writer) error
