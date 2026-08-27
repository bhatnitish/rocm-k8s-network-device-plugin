# Network Device Plugin Release Notes

## v1.2.1

### New Features

- **Multi-Version nicctl Bundling**: Container images now bundle multiple nicctl versions (up to 5) with automatic firmware detection at runtime. Eliminates compatibility issues when deploying across clusters with different NIC firmware versions.
- **OpenShift 4.21 Support**: Added compatibility with Red Hat OpenShift Container Platform 4.21.

### Improvements

- **Enhanced Firmware Detection**: Improved handling of partial NIC failures during firmware detection, ensuring plugin continues operation even when individual NICs fail.
- **Build Artifact Integrity**: Added checksum validation for downloaded nicctl binaries during image build.

### Bug Fixes

- **Command Execution Security**: Fixed shell injection vulnerability in command execution.
- **Partial NIC Failure Handling**: Resolved plugin crashes when firmware detection fails on individual NICs.

## v1.2.0

### New Features

- **Tech Support Diagnostics Tool**: Added comprehensive troubleshooting script (`tools/techsupport_dump.sh`) that collects Kubernetes resources, pod logs, and host-level diagnostics (PCI devices, RDMA links, nicctl output). Auto-detects deployment type (Helm/operator) and generates compressed tarball for support cases.

### Improvements

- **Helm Chart Support**: Helm charts are now published, enabling Kubernetes-native deployment and configuration.
- **Enhanced Container Capabilities**: Added python3-venv and IP utilities to Docker image for improved diagnostics and tooling support.
- **Helm Installation Guide**: Added comprehensive step-by-step documentation for Helm-based deployments with configuration examples.

### Bug Fixes

- **Init Container Startup**: Removed blocking Multus config check that prevented device plugin from starting when Multus wasn't configured as expected.
- **Configuration Parsing**: Fixed `excludeTopology` field location, moving it from selectors to top-level ResourceConfig for correct parsing.
- **Driver Detection**: Improved bare-metal vs VM environment detection to properly wait for required drivers (ionic + tawk_ipc/pds_core on bare metal).

## v1.1.0

- **nicctl Bundling**: `nicctl` binary (`1.117.5-a-56`) is now embedded in the Docker image, eliminating the need to mount nicctl from host.
- **Initial Pollara 400 Support**: Full integration with AMD Pollara 400 AI NICs including device discovery and resource allocation.
- **Developer Guide**: Added comprehensive developer documentation and contribution guidelines.
- **Test Coverage**: Integrated test coverage reporting into GitHub workflows.
- **Code Cleanup**: Removed unused code paths (CDI, auxnetDevice, accelerator) and deprecated Dockerfiles for improved maintainability.

## v1.0.0

- **AMD Network Device Plugin for Kubernetes**: Initial release enabling Kubernetes clusters to discover, manage, and allocate AMD Pollara AI NICs.
  - Automatic discovery of supported network devices with configurable selectors and resource naming.
  - Pod-level network device allocation via Kubernetes device plugin framework, preventing over-allocation.
  - Host-mounted nicctl support for device management and configuration.
  - Works across hypervisor, VM, and bare-metal environments.
