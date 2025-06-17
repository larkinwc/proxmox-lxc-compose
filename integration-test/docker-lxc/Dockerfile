# Dockerfile for LXC integration testing
FROM ubuntu:22.04

# Install LXC and dependencies
RUN apt-get update && apt-get install -y \
    lxc \
    lxc-utils \
    lxc-templates \
    bridge-utils \
    debootstrap \
    rsync \
    wget \
    curl \
    systemd \
    systemd-sysv \
    dbus \
    openssh-server \
    sudo \
    git \
    build-essential \
    iptables \
    ca-certificates \
    gnupg \
    lsb-release \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.23.4
RUN wget -q https://go.dev/dl/go1.23.4.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz && \
    rm go1.23.4.linux-amd64.tar.gz && \
    ln -s /usr/local/go/bin/go /usr/local/bin/go && \
    ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Ensure Go binaries are in PATH
ENV PATH="/usr/local/go/bin:${PATH}"

# Configure LXC
RUN echo 'lxc.net.0.type = veth' >> /etc/lxc/default.conf && \
    echo 'lxc.net.0.link = lxcbr0' >> /etc/lxc/default.conf && \
    echo 'lxc.net.0.flags = up' >> /etc/lxc/default.conf && \
    echo 'lxc.apparmor.profile = generated' >> /etc/lxc/default.conf && \
    echo 'lxc.apparmor.allow_nesting = 1' >> /etc/lxc/default.conf

# Create LXC bridge configuration
RUN mkdir -p /etc/default && \
    echo 'USE_LXC_BRIDGE="true"' > /etc/default/lxc-net && \
    echo 'LXC_BRIDGE="lxcbr0"' >> /etc/default/lxc-net && \
    echo 'LXC_ADDR="10.0.3.1"' >> /etc/default/lxc-net && \
    echo 'LXC_NETMASK="255.255.255.0"' >> /etc/default/lxc-net && \
    echo 'LXC_NETWORK="10.0.3.0/24"' >> /etc/default/lxc-net && \
    echo 'LXC_DHCP_RANGE="10.0.3.2,10.0.3.254"' >> /etc/default/lxc-net && \
    echo 'LXC_DHCP_MAX="253"' >> /etc/default/lxc-net

# Create test user
RUN useradd -m -s /bin/bash testuser && \
    echo 'testuser:testpass' | chpasswd && \
    usermod -aG sudo testuser && \
    echo 'testuser ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# Create SSH key for remote access
RUN mkdir -p /home/testuser/.ssh && \
    ssh-keygen -t rsa -b 2048 -f /home/testuser/.ssh/id_rsa -N "" && \
    cp /home/testuser/.ssh/id_rsa.pub /home/testuser/.ssh/authorized_keys && \
    chown -R testuser:testuser /home/testuser/.ssh && \
    chmod 700 /home/testuser/.ssh && \
    chmod 600 /home/testuser/.ssh/*

# Configure SSH
RUN mkdir -p /var/run/sshd && \
    echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config && \
    echo 'PasswordAuthentication yes' >> /etc/ssh/sshd_config && \
    echo 'PubkeyAuthentication yes' >> /etc/ssh/sshd_config

# Create startup script
COPY start-services.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/start-services.sh

# Create test directory
RUN mkdir -p /opt/lxc-compose-test
WORKDIR /opt/lxc-compose-test

EXPOSE 22

CMD ["/usr/local/bin/start-services.sh"]