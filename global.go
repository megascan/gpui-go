package gpui

// Global returns the global of type G; it panics if no global of that type has been set.
// It is upstream's ReadGlobal::global; any type can be a global (the Global marker trait is
// dropped) and UpdateGlobal's methods are the package functions UpdateGlobal, UpdateGlobalR
// and SetGlobal.
func Global[G any](cx *App) *G {
	return cx.Global[G]()
}
