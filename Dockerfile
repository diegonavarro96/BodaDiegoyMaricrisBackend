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
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/wedding-rsvp /app/wedding-rsvp

# Railway mounts the persistent volume at /data. Pre-create it so the
# image is also runnable standalone, and so the app user owns it.
RUN mkdir -p /data && chown -R app:app /data /app
USER app

EXPOSE 8080
CMD ["/app/wedding-rsvp"]
