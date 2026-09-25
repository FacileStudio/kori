package chat

import (
	"context"
	"errors"
	"strings"
)

// VerificationStatus reports whether the account has cross-signing keys
// on the homeserver and whether this device is already signed by them.
func (m *Matrix) VerificationStatus(ctx context.Context) (bool, bool, error) {
	if m.helper == nil {
		return false, false, errors.New("matrix: crypto helper not initialized")
	}
	mach := m.helper.Machine()
	if mach == nil {
		return false, false, errors.New("matrix: olm machine not initialized")
	}
	return mach.GetOwnVerificationStatus(ctx)
}

// VerifyWithRecoveryKey verifies this device by downloading the cross-signing
// keys from SSSS using the given recovery key, and signing this device.
func (m *Matrix) VerifyWithRecoveryKey(ctx context.Context, recoveryKey string) error {
	if m.helper == nil {
		return errors.New("matrix: crypto helper not initialized")
	}
	mach := m.helper.Machine()
	if mach == nil {
		return errors.New("matrix: olm machine not initialized")
	}
	trimmed := strings.TrimSpace(recoveryKey)
	if trimmed == "" {
		return errors.New("matrix: recovery key is empty")
	}
	return mach.VerifyWithRecoveryKey(ctx, trimmed)
}
