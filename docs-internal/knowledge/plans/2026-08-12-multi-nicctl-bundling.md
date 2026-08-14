# Multi-Version nicctl Bundling - Complete Plan

**Date**: 2026-08-12  
**Author**: Yuva Shankar  
**Status**: Design Finalized, Ready for Implementation  

---

## Executive Summary

**Problem**: Need to support multiple AINIC firmware versions in a single Docker image.

**Solution**: Bootstrap latest version (uncompressed, 73 MB) + compress older versions (8.4 MB each)

**Size Impact**:
- Current (1 version): 73 MB
- Typical (2 versions): 81.4 MB (+8.4 MB)
- Maximum (3 versions): 89.8 MB (+16.8 MB)
- **Hard limit**: 3 versions per image

**How It Works**:
1. Latest version uncompressed for fast firmware detection
2. Older versions xz-compressed (98% size reduction)
3. Entrypoint detects firmware via `nicctl-bootstrap show firmware`
4. Auto-decompresses matched version to `/usr/sbin/nicctl`
5. Zero overhead after startup

**Key Benefits**:
- ✅ Auto-detection from NIC hardware
- ✅ Small overhead: 8.4 MB per additional version
- ✅ Fast startup: 100ms typical, 1-2sec worst case
- ✅ Graceful fallback to latest if version not found

**Quick Usage**:
```bash
docker build \
  --build-arg AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77" \
  --build-arg BOOTSTRAP_VERSION="1.117.5-a-77" \
  -f images/Dockerfile \
  -t sriov-device-plugin:multi .
```

---

## Table of Contents

1. [Overview](#overview)
2. [Size Analysis](#size-analysis)
3. [Design Decisions](#design-decisions)
4. [Architecture](#architecture)
5. [Implementation Specification](#implementation-specification)
6. [Testing Strategy](#testing-strategy)
7. [Appendix: Research & Analysis](#appendix-research--analysis)

---

## Overview

### Objective

Enable bundling multiple nicctl binary versions within a single Docker image to support different AINIC firmware versions without requiring separate container images or host-mounted binaries.

### Background

**Current State:**
- nicctl binary bundled in Docker image (as of v1.1.0)
- Single version per image: `1.117.5-a-56` (v1.1.0) or `1.117.5-a-77` (v1.2.0)
- Binary size: **526 MB** uncompressed, **73 MB** stripped
- Problem: Supporting multiple firmware versions requires multiple container images

**Motivation:**
- Users may have mixed AINIC firmware versions across cluster nodes
- Simplify deployment by supporting multiple versions in one image
- Reduce maintenance burden of multiple image variants

### Key Requirements

1. ✅ Auto-detect firmware version from NIC hardware
2. ✅ Support 2+ nicctl versions in single image
3. ✅ Minimal image size overhead
4. ✅ Zero runtime performance impact after startup
5. ✅ Graceful fallback when firmware version not available

---

## Size Analysis

### Compression Comparison

| Variant | Size per binary | Reduction |
|---------|-----------------|-----------|
| Original (uncompressed) | 526 MB | baseline |
| Stripped | 73 MB | 86% |
| Stripped + gzip | 15 MB | 97% |
| **Stripped + xz** | **8.4 MB** | **98%** ⭐ |

### Multi-Version Projections

**For 2 versions** (current requirement: a-56, a-77):
- All uncompressed: 1.05 GB ❌
- All stripped: 146 MB ⚠️
- Bootstrap + compressed: **81.4 MB** ✅

**For 3 versions** (maximum allowed):
- All uncompressed: 1.57 GB ❌
- All stripped: 219 MB ⚠️
- Bootstrap + compressed: **89.8 MB** ✅

**Maximum Bundle Limit:** 3 versions per image
- Prevents excessive image bloat
- Keeps image size under 100 MB
- Typically covers current + 2 previous firmware versions

**Selected Approach:** Bootstrap latest (73 MB) + compress older versions (8.4 MB each)

---

## Design Decisions

### Decision 1: Detection Method ✅

**Use `nicctl show firmware` to read actual firmware from hardware**

**Rationale:**
- nicctl tool version (`nicctl --version`) is NOT reliable for compatibility
- Must match firmware version exactly for proper functionality
- Firmware version is the source of truth

**Hardware Test Results** (from root@10.9.11.245):

```bash
$ nicctl --version
1.117.5-a-77
# Performance: 11ms (no hardware access)

$ nicctl show firmware
NIC : 42424650-4c32-3531-3830-303230000000 (0000:41:00.0)
Boot0                          : 21
CPLD                           : 3.18 (primary)
Firmware-B                     : 1.117.5-a-77
...
# Performance: 109ms (hardware access)
```

**Parsing:**
```bash
FIRMWARE_VER=$(nicctl-bootstrap show firmware 2>/dev/null | \
    grep -E "^Firmware-[AB]" | awk '{print $3}' | head -1)
```

### Decision 2: No Exact Match → Use Most Recent ✅

**If detected firmware version not in bundled set, use the most recent bundled version**

**Rationale:**
- Newer nicctl versions typically backward compatible
- Better to run with recent version than fail
- Log warning for operator visibility

### Decision 3: Single Firmware Per Host ✅

**All NICs on same host have same firmware version**

**Rationale:**
- Operational policy ensures uniform firmware per host
- Simplifies detection (use first NIC only)
- No need to handle mixed versions

### Decision 4: No Hardware → Use Most Recent ✅

**If `nicctl show firmware` fails (no NICs), use most recent bundled version**

**Rationale:**
- Allows container to start in CI/VM environments
- Same fallback as "no exact match" scenario
- Log warning for debugging

### Decision 5: Entrypoint Detection, Not Wrapper ✅

**Use entrypoint script that runs ONCE at container startup**

**Why not wrapper on every nicctl call:**
- ✅ Better performance: After startup, all nicctl calls are direct (no wrapper overhead)
- ✅ Simpler debugging: `ps` shows actual nicctl process
- ✅ Cleaner architecture: Setup once at init, not on every invocation
- ✅ More transparent: Users interact with real nicctl binary

**Trade-off:** Slightly slower container startup (1-2 seconds max) vs every invocation

### Decision 6: Maximum 3 Versions, Explicit Bootstrap ✅

**Limit bundle to maximum 3 nicctl versions**

**Rationale:**
- Prevents image bloat (93 MB for 3 vs 113 MB for 5)
- 5 versions = current + 4 previous (covers extended compatibility)
- Forces periodic cleanup of very old versions

**Bootstrap (Latest) Selection:**
- **Explicit**: Use `BOOTSTRAP_VERSION` build arg (recommended)
- **Implicit**: If not specified, use last version in `AINIC_VERSIONS` list

**Version Format**: `major.minor.patch-variant-build`
- Example: `1.117.5-a-77` where `77` is build number
- Higher build number = newer version
- Format must be consistent for parsing

**Validation:** Build fails if more than 5 versions specified

```bash
# Recommended: Explicit bootstrap
--build-arg AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77,1.117.5-a-88"
--build-arg BOOTSTRAP_VERSION="1.117.5-a-88"  # Latest

# Fallback: Last in list is bootstrap
--build-arg AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77,1.117.5-a-88"
# "1.117.5-a-88" becomes bootstrap
```

---

## Architecture

### Image Structure

```
┌──────────────────────────────────────────────┐
│ Docker Image                                 │
│                                              │
│ /usr/sbin/nicctl-bootstrap (73 MB)          │
│   └── Latest version (1.117.5-a-77)         │
│       Uncompressed for fast detection       │
│                                              │
│ /opt/nicctl-versions/                        │
│   └── nicctl-1.117.5-a-56.xz  (8.4 MB)     │
│       Older versions, xz compressed          │
│                                              │
│ /opt/bootstrap-version.txt                   │
│   └── Contains: "1.117.5-a-77"              │
│                                              │
│ /entrypoint.sh                               │
│   └── Detection + decompression logic       │
└──────────────────────────────────────────────┘
```

### Runtime Flow

```
Container Start
     │
     ▼
/entrypoint.sh runs
     │
     ▼
nicctl-bootstrap show firmware
     │
     ├─── NICs detected ────────┐
     │                          │
     ├─── No NICs ───────┐      │
     │                   │      │
     ▼                   ▼      ▼
Parse output      Use latest  Extract version
     │                   │      │
     └─────────┬─────────┘      │
               │                │
               ▼                │
     Check if version bundled   │
               │                │
     ├─── Found ────────────────┘
     │                          │
     ├─── Not Found ─────┐      │
     │                   │      │
     ▼                   ▼      ▼
Is bootstrap?       Use latest  Decompress
     │                   │      │
     ├─Yes──┐            │      │
     │      │            │      │
     ▼      │            │      │
Symlink     │            │      │
     │      │            │      │
     └──┬───┴────────────┴──────┘
        │
        ▼
/usr/sbin/nicctl ready
        │
        ▼
exec /usr/bin/sriovdp
```

### Startup Performance

| Scenario | Overhead | Frequency |
|----------|----------|-----------|
| Bootstrap match | ~100 ms | Common (latest firmware) |
| Decompress needed | 1-2 sec | Less common (older firmware) |
| No hardware | ~100 ms | Rare (CI/VM only) |

---

## Implementation Approach

### Build-Time Steps

**Phase 1: Multi-Stage Docker Build**

1. **Validation Stage**
   - Validate `AINIC_VERSIONS` not empty
   - Validate maximum 3 versions
   - Validate `BOOTSTRAP_VERSION` in version list (if specified)
   - Determine bootstrap version (explicit or last in list)

2. **Bootstrap Binary Stage**
   - Install bootstrap nicctl version from repository
   - Strip binary (526 MB → 73 MB)
   - Save as `/usr/sbin/nicctl-bootstrap`
   - Record bootstrap version in `/opt/bootstrap-version.txt`

3. **Compressed Versions Stage**
   - For each non-bootstrap version:
     - Install from repository
     - Strip binary (526 MB → 73 MB)
     - Compress with xz (73 MB → 8.4 MB)
     - Save to `/opt/nicctl-versions/nicctl-{VERSION}.xz`

4. **Final Image Assembly**
   - Copy bootstrap binary (uncompressed)
   - Copy compressed versions
   - Copy shared libraries (libpci)
   - Install xz-utils for runtime decompression
   - Copy entrypoint script

**Implementation Reference**: See `images/Dockerfile` (nicctlbuilder stage)

### Runtime Steps (Entrypoint)

**Phase 2: Container Startup Detection**

1. **Detect Firmware Version**
   ```
   Run: nicctl-bootstrap show firmware
   Parse: Firmware-A/B field → version string
   Fallback: Use bootstrap version if no NICs
   ```

2. **Select Binary**
   ```
   If detected == bootstrap: Use bootstrap directly (symlink)
   Else if compressed exists: Decompress to /usr/sbin/nicctl
   Else: Fallback to bootstrap (log warning)
   ```

3. **Verify & Start**
   ```
   Verify: nicctl --version works
   Log: Ready with version
   Exec: Start main container process (sriovdp)
   ```

**Implementation Reference**: See `images/nicctl-setup.sh`

### Build Configuration

**Typical Build (2 versions):**
```bash
docker build \
  --build-arg AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77" \
  --build-arg BOOTSTRAP_VERSION="1.117.5-a-77" \
  -f images/Dockerfile \
  -t ghcr.io/pensando/sriov-device-plugin:v1.3.0 .
```

**Maximum (5 versions):**
```bash
docker build \
  --build-arg AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77,1.117.5-a-88,1.117.5-a-103,1.117.5-a-147" \
  --build-arg BOOTSTRAP_VERSION="1.117.5-a-147" \
  -f images/Dockerfile \
  -t ghcr.io/pensando/sriov-device-plugin:v1.3.0 .
```

**Via Makefile:**
```bash
# Default (2 versions)
make image

# Custom versions
make image DOCKERARGS="--build-arg AINIC_VERSIONS=1.117.5-a-147"
make image DOCKERARGS="--build-arg AINIC_VERSIONS=1.117.5-a-56,1.117.5-a-77,1.117.5-a-88,1.117.5-a-103,1.117.5-a-147 --build-arg BOOTSTRAP_VERSION=1.117.5-a-147"
```

**Version Ordering Guidelines:**

1. **Order doesn't matter** for compressed versions (all treated equally)
2. **Bootstrap version** is what matters (must be specified or last in list)
3. **Recommended practice**: List versions in chronological order (oldest to newest)
   ```
   AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77,1.117.5-a-88"
                   ↑ oldest      ↑ middle     ↑ newest (bootstrap)
   ```
4. **Validation**: Build fails if:
   - Zero versions specified
   - More than 3 versions specified
   - `BOOTSTRAP_VERSION` not in `AINIC_VERSIONS` list

**Build Validation Examples:**

```bash
# ✅ Valid: 2 versions, explicit bootstrap
AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77"
BOOTSTRAP_VERSION="1.117.5-a-77"

# ✅ Valid: 3 versions (maximum)
AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77,1.117.5-a-88"
BOOTSTRAP_VERSION="1.117.5-a-88"

# ✅ Valid: Implicit bootstrap (last in list)
AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77"
# Bootstrap = 1.117.5-a-77

# ❌ Invalid: Too many versions
AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77,1.117.5-a-88,1.117.5-a-99"
# ERROR: Maximum 3 versions allowed, got 4

# ❌ Invalid: Bootstrap not in list
AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77"
BOOTSTRAP_VERSION="1.117.5-a-88"
# ERROR: BOOTSTRAP_VERSION '1.117.5-a-88' not in AINIC_VERSIONS

# ❌ Invalid: Empty list
AINIC_VERSIONS=""
# ERROR: AINIC_VERSIONS cannot be empty
```

---

## Example Scenarios

### Scenario 1: Bootstrap Match (Typical)
**Host firmware**: 1.117.5-a-77 (latest)
- Uses bootstrap directly (no decompression needed)
- Startup overhead: ~100ms
- Most common case in production

### Scenario 2: Older Version Decompression
**Host firmware**: 1.117.5-a-56 (older)
- Decompresses `nicctl-1.117.5-a-56.xz` to `/usr/sbin/nicctl`
- Startup overhead: 1-2 seconds (one-time)
- Subsequent restarts use same version

### Scenario 3: Version Not Bundled
**Host firmware**: 1.117.5-a-88 (newer, not in image)
- WARNING logged to container logs
- Falls back to bootstrap (1.117.5-a-77)
- Startup overhead: ~100ms
- Relies on backward compatibility

### Scenario 4: No Hardware (CI/VM)
**Environment**: No NICs present (testing/CI)
- `nicctl show firmware` fails gracefully
- WARNING logged
- Uses bootstrap version
- Startup overhead: ~100ms
- Allows container to start in test environments

---

## Testing Strategy

### Unit Tests

- [ ] Entrypoint script parsing logic
- [ ] Version extraction from `show firmware` output
- [ ] Symlink creation for bootstrap version
- [ ] Decompression logic for older versions
- [ ] Fallback behavior when version not found

### Integration Tests

**Test 1: Bootstrap version match**
```bash
# Host with firmware 1.117.5-a-77 (latest)
docker run --privileged test-image
# Expected: No decompression, uses bootstrap directly
# Startup: ~100ms overhead
```

**Test 2: Older version decompression**
```bash
# Host with firmware 1.117.5-a-56 (older)
docker run --privileged test-image
# Expected: Decompresses a-56 version
# Startup: ~1-2sec overhead
```

**Test 3: Version not bundled**
```bash
# Host with firmware 1.117.5-a-88 (not in image)
docker run --privileged test-image
# Expected: WARNING logged, uses bootstrap
# Startup: ~100ms overhead
```

**Test 4: No hardware (CI/VM)**
```bash
# Environment without NICs
docker run test-image
# Expected: WARNING logged, uses bootstrap
# Startup: ~100ms overhead
```

**Test 5: Multi-NIC same firmware**
```bash
# Host with 2+ NICs, same firmware
docker run --privileged test-image
# Expected: Uses first NIC's firmware version
```

### Performance Tests

- [ ] Measure actual decompression time on target hardware
- [ ] Verify startup overhead < 2 seconds in worst case
- [ ] Confirm zero overhead after entrypoint completes
- [ ] Check memory usage with cached binaries

### Validation Checklist

- [ ] All bundled versions execute successfully
- [ ] Firmware detection works on real hardware
- [ ] Decompression creates functional binary
- [ ] Fallback to bootstrap works correctly
- [ ] Warnings logged in all fallback cases
- [ ] Container exits cleanly if nicctl not functional
- [ ] Image size within expected range (< 100 MB for 2 versions)
- [ ] Stripped binary has identical behavior to original

---

## Success Criteria

1. ✅ Single Docker image supports 2+ nicctl versions
2. ✅ Image size < 100 MB for 2 versions (target: 81.4 MB)
3. ✅ **Auto-detection**: Automatically detects firmware via `nicctl show firmware`
4. ✅ **Fast startup**: < 2 seconds overhead in worst case
5. ✅ Zero runtime performance impact after startup
6. ✅ All bundled versions functional
7. ✅ Graceful fallback when firmware version not available
8. ✅ Clear warnings logged for debugging
9. ✅ Documentation updated

---

## Timeline

- **Planning**: 2026-08-12 ✅ Complete
- **Implementation**: TBD
  - Update Dockerfile
  - Create entrypoint script
  - Update existing Dockerfile to reference multi-version approach
- **Testing**: TBD
  - Hardware validation
  - CI integration
  - Performance benchmarks
- **Documentation**: TBD
  - Update README.md
  - Release notes
  - Helm chart documentation
- **Release**: Target v1.3.0

---

## Appendix: Key Research Findings

### Compression Analysis

Tested compression methods on actual nicctl binary (526 MB original):

| Method | Size | Reduction | Notes |
|--------|------|-----------|-------|
| Stripped | 73 MB | 86% | Removes debug symbols |
| Stripped+gzip | 15 MB | 97% | Faster decompress |
| **Stripped+xz** | **8.4 MB** | **98%** | **Selected** - best compression |

**Selected xz** for best size/performance ratio

### Detection Method Validation

**Hardware Test**: root@10.9.11.245

**`nicctl --version`** (11ms, no hardware access):
- Returns: `1.117.5-a-77`
- **Not used**: Shows tool version, not firmware version

**`nicctl show firmware`** (109ms, requires NICs):
- Returns: Multi-NIC output with `Firmware-A/B` fields
- **Selected**: Authoritative firmware version source
- Parsing: Extract from `Firmware-[AB]` field

### Alternatives Considered (Not Selected)

**Wrapper Script Approach**:
- Every nicctl call goes through wrapper
- Rejected: Performance overhead, debugging complexity

**Eager Decompression**:
- Decompress all versions at startup
- Rejected: Wastes memory, slow startup

**sysfs Detection**:
- Read firmware from `/sys/bus/pci/devices/`
- Future optimization: Not validated yet

---

## References

- **Related PRs**:
  - #26: Original nicctl bundling
  - #50: Unified AINIC support (Salina/Vulcano)

- **Implementation files**:
  - `images/Dockerfile` - Multi-version Dockerfile
  - `images/entrypoint.sh` - Detection and setup script
  - `docs/nicctl-bundling-analysis.md` - Compression analysis

---

**End of Plan**
