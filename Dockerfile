FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.24.3-alpine3.21 AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git=2.47.2-r0

# Copy the current directory contents into the container
COPY . .

# Build the application
# RUN --mount=type=cache,target=/go/pkg \
#     --mount=type=cache,target=/root/.cache/go-build \
#     ./build.sh build_docker
RUN ./build.sh build_docker

# Minimal passwd entry for non-privileged user
RUN echo "pixivfe:x:10001:10001::/:/" >> /etc/passwd

# Stage for a smaller final image
FROM scratch

# Copy necessary files from the builder stage
COPY --from=builder /app/pixivfe /app/pixivfe
COPY --from=builder /app/assets /app/assets
COPY --from=builder /app/i18n /app/i18n
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd

# Set the working directory
WORKDIR /app

# Expose port 8282
EXPOSE 8282

# Switch to non-privileged user
USER pixivfe

# Set the entrypoint to the binary name
ENTRYPOINT ["/app/pixivfe"]
