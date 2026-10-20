package repogov

import (
	"errors"
	"syscall"
)

// errSharingViolation is the Windows ERROR_SHARING_VIOLATION code, which
// the syscall package does not name.
const errSharingViolation syscall.Errno = 32

// isTransientRenameError reports whether a rename failed because another
// process holds the destination open (ERROR_ACCESS_DENIED or
// ERROR_SHARING_VIOLATION), a condition that clears once the handle closes.
func isTransientRenameError(err error) bool {
	return errors.Is(err, syscall.ERROR_ACCESS_DENIED) || errors.Is(err, errSharingViolation)
}
