/*
Copyright (c) Advanced Micro Devices, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package deviceconfig

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestExecWithContextNoShellInjection is a regression guard for KUBE-40 (SEC-00393).
//
// intfUUID flows from nicctl JSON output into the queue-pair clear command. The
// original implementation built a string and ran it via `/bin/sh -c`, so any shell
// metacharacter in the lif value yielded arbitrary command execution as root in a
// privileged DaemonSet. execWithContext now takes an argv, so metacharacters must be
// passed as literal arguments and never interpreted by a shell.
//
// The malicious payload embeds a `; touch <marker>` sequence. If a shell ever
// interprets it, the marker file is created and the test fails.
func TestExecWithContextNoShellInjection(t *testing.T) {
	dc := NewDevConfigClient()

	marker := filepath.Join(t.TempDir(), "pwned")
	maliciousLif := "abc; touch " + marker

	// New, safe form: mirrors setupDevice's call. /bin/echo stands in for nicctl so
	// the test runs without the real binary. The payload is a single argv element.
	_, err := dc.execWithContext(configClientTimeoutInSecs, "/bin/echo",
		"clear", "rdma", "internal", "queue-pair", "--lif", maliciousLif)
	if err != nil {
		t.Fatalf("execWithContext returned error: %v", err)
	}

	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatalf("SECURITY REGRESSION: marker %q was created — shell injection is possible", marker)
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("unexpected error stat-ing marker: %v", statErr)
	}
}

// TestShellInjectionPositiveControl proves the test above can actually detect the
// vulnerability: it reproduces the ORIGINAL `/bin/sh -c` behavior with the same
// payload and asserts that the injection fires (marker IS created). If this ever
// fails, the test harness itself is broken and the negative test is worthless.
func TestShellInjectionPositiveControl(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "pwned")
	maliciousLif := "abc; touch " + marker

	ctx, cancel := context.WithTimeout(context.Background(), configClientTimeoutInSecs*time.Second)
	defer cancel()

	// Original vulnerable form, inlined here on purpose.
	vulnCmd := "echo clear rdma internal queue-pair --lif " + maliciousLif
	if err := exec.CommandContext(ctx, "/bin/sh", "-c", vulnCmd).Run(); err != nil {
		t.Fatalf("positive-control command failed to run: %v", err)
	}

	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatalf("positive control did not trigger injection (test harness is broken): %v", statErr)
	}
}
