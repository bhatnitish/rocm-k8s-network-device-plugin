## Dockerfile Build

This directory contains the Dockerfile and scripts for building the AMD Network Device Plugin container image.

### Building the Image

Typically you'd build this from the root of your repository, using `make image`:

```bash
make image
```

This builds with default nicctl versions (`1.117.5-a-77` and `1.117.5-a-147`).

Or use docker directly:

```bash
docker build -t ghcr.io/pensando/sriov-network-device-plugin -f ./images/Dockerfile .
```

### Multi-Version nicctl Support

The image supports bundling multiple nicctl versions for cross-firmware compatibility. The container automatically detects the NIC firmware version at startup and selects the matching binary.

**Build Arguments:**

- `AINIC_VERSIONS`: Comma-separated list of nicctl versions to bundle (max 5)
  - Default: `1.117.5-a-77,1.117.5-a-147`
  - Example: `--build-arg AINIC_VERSIONS="1.117.5-a-56,1.117.5-a-77,1.117.5-a-147"`

- `BOOTSTRAP_VERSION`: Version to install uncompressed (must be in AINIC_VERSIONS)
  - Default: Last version in AINIC_VERSIONS list
  - Example: `--build-arg BOOTSTRAP_VERSION="1.117.5-a-147"`

**Examples:**

Single-version build:
```bash
make image DOCKERARGS="--build-arg AINIC_VERSIONS=1.117.5-a-147"
```

Multi-version with custom versions:
```bash
make image DOCKERARGS="--build-arg AINIC_VERSIONS=1.117.5-a-56,1.117.5-a-77,1.117.5-a-88 --build-arg BOOTSTRAP_VERSION=1.117.5-a-88"
```

**Image Size Reference:**
- 1 version: ~73 MB
- 2 versions: ~93 MB (default)
- 5 versions (max): ~113 MB

Each additional version adds ~10 MB (compressed with xz).

### Runtime Firmware Detection

The **nicctl-setup.sh** script handles version selection at container startup:

1. For single-version builds: passes directly to entrypoint.sh
2. For multi-version builds:
   - Detects NIC firmware version via `nicctl-bootstrap show firmware`
   - Selects matching binary from bundled versions
   - Decompresses on-demand if needed
   - Falls back to bootstrap if no exact match found

---

## Daemonset Deployment

You may wish to deploy SR-IOV device plugin as a daemonset, you can do so by starting with the example Daemonset shown here:

```
$ kubectl create -f ./deployments/k8s-v1.16/sriovdp-daemonset.yaml
```
> For K8s version v1.15 or older use `deployments/k8s-v1.10-v1.15/sriovdp-daemonset.yaml` instead.

Note: The likely best practice here is to build your own image given the Dockerfile, and then push it to your preferred registry, and change the `image` fields in the Daemonset YAML to reference that image.

---

### Development notes

Example docker run command:

```
$ docker run -it -v /var/lib/kubelet/device-plugins:/var/lib/kubelet/device-plugins -v /var/lib/kubelet/plugins_registry:/var/lib/kubelet/plugins_registry -v /sys/class/net:/sys/class/net --entrypoint=/bin/bash ghcr.io/k8snetworkplumbingwg/sriov-network-device-plugin
```

Originally inspired by and is a portmanteau of the [Flannel daemonset](https://github.com/coreos/flannel/blob/master/Documentation/kube-flannel.yml), the [Calico Daemonset](https://github.com/projectcalico/calico/blob/master/v2.0/getting-started/kubernetes/installation/hosted/k8s-backend-addon-manager/calico-daemonset.yaml), and the [Calico CNI install bash script](https://github.com/projectcalico/cni-plugin/blob/be4df4db2e47aa7378b1bdf6933724bac1f348d0/k8s-install/scripts/install-cni.sh#L104-L153).
