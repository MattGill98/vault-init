# Builder image
FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git
ENV USER=appuser
ENV UID=10001
# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"
WORKDIR $GOPATH/src/mypackage/myapp/
COPY . .
RUN go get -d -v
RUN go build -ldflags="-w -s" -o /go/bin/vault-init

# Runtime image
FROM scratch

# Import the user and group files from the builder.
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy our static executable.
COPY --from=builder /go/bin/vault-init /go/bin/vault-init

# Use an unprivileged user.
USER appuser:appuser

# Enable Kubernetes storage
ENV KUBERNETES_STORAGE=true

# Run the hello binary.
ENTRYPOINT ["/go/bin/vault-init"]
