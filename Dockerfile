# syntax=docker/dockerfile:1.6

# --- build stage -----------------------------------------------------------
FROM golang:1.23-alpine AS build
WORKDIR /src

# Project uses only the Go standard library, so there's no go.sum.
# If deps get added later, add `COPY go.sum ./` above the source copy.
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath -ldflags='-s -w' \
      -o /out/wedding-rsvp .

# --- runtime stage ---------------------------------------------------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/wedding-rsvp /app/wedding-rsvp

# Run as root inside the container. Reason: Railway mounts the persistent
# volume at /data at runtime and the mount replaces in-image ownership, so
# a non-root user can't write to /data without an entrypoint that chowns
# at boot. Container isolation provides the security boundary here, not
# in-container UID separation.
RUN mkdir -p /data

EXPOSE 8080
CMD ["/app/wedding-rsvp"]
