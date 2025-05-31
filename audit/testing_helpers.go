// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Helpers used for testing
*/
package audit

import (
	"testing"

	"go.uber.org/zap"
)

// NewTestingLogger creates and initializes GlobalAuditor.Logger for use in tests.
func NewTestingLogger(t *testing.T) {
	t.Helper()

	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatal(err)
	}

	GlobalAuditor = &Auditor{
		Logger: logger.Sugar(),
	}
}
