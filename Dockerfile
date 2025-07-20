# GoReleaser optimized Dockerfile
# This Dockerfile is designed to work with GoReleaser's build context
# The binary is pre-built by GoReleaser and copied from the build context

FROM gcr.io/distroless/static:nonroot

# Copy the pre-built binary from GoReleaser's build context
COPY do-firewall-allowlister /usr/local/bin/do-firewall-allowlister

# Use non-root user
USER nonroot:nonroot

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/do-firewall-allowlister"]

# Default command
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="DigitalOcean Firewall Allowlister"
LABEL org.opencontainers.image.description="Automatically manages DigitalOcean firewall rules with supported sources"
LABEL org.opencontainers.image.vendor="kholisrag"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/kholisrag/do-firewall-allowlister"