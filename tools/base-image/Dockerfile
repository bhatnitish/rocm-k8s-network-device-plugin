ARG BUILD_BASE_IMAGE=ubuntu:22.04
FROM ${BUILD_BASE_IMAGE}

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# remove old version of go
RUN rm -rf /usr/local/go

RUN apt-get update && apt-get install -y wget protobuf-compiler \
    curl locales ca-certificates build-essential git vim \
    net-tools sudo cmake build-essential && \
    install -m 0755 -d /etc/apt/keyrings && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

RUN localedef -i en_US -f UTF-8 en_US.UTF-8

# download docker
RUN curl --proto '=https' --tlsv1.2 -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc && \
    chmod a+r /etc/apt/keyrings/docker.asc

RUN echo 'deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu jammy stable' > /etc/apt/sources.list.d/docker.list

RUN apt-get update && apt-get install -y \
  docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin \
  make python3-sphinx python3-pip \
  git && apt-get clean && rm -rf /var/lib/apt/lists/*

# download go1.23.4 (sha256 pinned)
RUN wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz && \
    echo "6924efde5de86fe277676e929dc9917d466efa02fb934197bc2eba35d5680971  go1.23.4.linux-amd64.tar.gz" | sha256sum -c - && \
    tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz && \
    rm go1.23.4.linux-amd64.tar.gz

# download and install kubectl (sha256 pinned)
RUN curl --proto '=https' --tlsv1.2 -fsSLo /usr/local/bin/kubectl 'https://dl.k8s.io/release/v1.30.4/bin/linux/amd64/kubectl' && \
    echo "2ffd023712bbc1a9390dbd8c0c15201c165a69d394787ef03eda3eccb4b9ac06  /usr/local/bin/kubectl" | sha256sum -c - && \
    chmod +x /usr/local/bin/kubectl

# download and install helm (pinned release + sha256 verified)
RUN curl --proto '=https' --tlsv1.2 -fsSL -o helm.tar.gz https://get.helm.sh/helm-v3.21.4-linux-amd64.tar.gz && \
    echo "61f88ab166748cb19604d7884cb100ae9ccb13804ddeb98e08af167eacbb6a14  helm.tar.gz" | sha256sum -c - && \
    tar -xzf helm.tar.gz --strip-components=1 -C /usr/local/bin linux-amd64/helm && \
    rm helm.tar.gz

ARG INSECURE_REGISTRY
RUN echo "INSECURE_REGISTRY is: $INSECURE_REGISTRY" && \
    if [ -n "$INSECURE_REGISTRY" ]; then \
    mkdir -p /etc/docker && \
    echo "{ \
    \"insecure-registries\": [\"$INSECURE_REGISTRY\"] \
    }" > /etc/docker/daemon.json; \
    else \
    echo "INSECURE_REGISTRY is not set"; \
    fi

ADD ./entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
