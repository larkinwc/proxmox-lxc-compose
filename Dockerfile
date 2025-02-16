# Using a specific version of Alpine for better reproducibility
FROM alpine:3.19

# Add labels that will be populated by GoReleaser
LABEL org.opencontainers.image.source="https://github.com/larkinwc/proxmox-lxc-compose"
LABEL org.opencontainers.image.description="A docker-compose like tool for managing LXC containers in Proxmox"
LABEL org.opencontainers.image.licenses="MIT"

# Install dependencies
RUN apk add --no-cache \
    lxc \
    docker-cli

# Create a non-root user and group
RUN addgroup -S lxccompose && adduser -S lxccompose -G lxccompose

# Copy the binary from the GoReleaser build
COPY lxc-compose /usr/local/bin/
RUN chmod +x /usr/local/bin/lxc-compose

# Use the non-root user
USER lxccompose

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/lxc-compose"]
# Provide a default command that can be overridden
CMD ["--help"]