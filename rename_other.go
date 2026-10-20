//go:build !windows

package repogov

// isTransientRenameError reports whether a rename failure may clear on
// retry. Outside Windows, rename replaces a file other processes hold open,
// so no failure is transient.
func isTransientRenameError(error) bool {
	return false
}
