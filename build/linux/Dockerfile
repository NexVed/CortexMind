# CortexMind Linux builder — Ubuntu 22.04 (glibc 2.35) so the AppImage runs on
# most current distros. Wails/WebKitGTK cannot be cross-compiled from Windows;
# this image is the supported way to produce CortexMind-*-x86_64.AppImage.
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive \
    APPIMAGE_EXTRACT_AND_RUN=1 \
    CGO_ENABLED=1 \
    GOPATH=/go \
    PATH=/usr/local/go/bin:/go/bin:/usr/local/node/bin:${PATH} \
    GO_VERSION=1.25.0 \
    NODE_VERSION=20.18.1

RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        curl \
        file \
        git \
        libgtk-3-dev \
        libwebkit2gtk-4.1-dev \
        desktop-file-utils \
        pkg-config \
        xz-utils \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
      | tar -C /usr/local -xz

RUN curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" \
      | tar -C /usr/local -xJ \
    && mv /usr/local/node-v${NODE_VERSION}-linux-x64 /usr/local/node

WORKDIR /src
# bash -c so a Windows CRLF checkout cannot break the shebang.
CMD ["bash", "-c", "sed -i 's/\\r$//' /src/build/linux/entrypoint.sh && exec bash /src/build/linux/entrypoint.sh"]
