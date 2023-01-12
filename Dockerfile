FROM golang:1.19.5-alpine3.17 as builder

ARG BIN=rook-block-be-gone
RUN apk --update --no-cache add \
    binutils \
    && rm -rf /root/.cache
WORKDIR /go/src/github.com/jhoblitt/rook-block-be-gone
COPY . .
RUN go build -v ./... \
    && strip "$BIN"

FROM alpine:3.17
RUN apk --update --no-cache add \
    bash \
    ca-certificates \
    tzdata \
    && rm -rf /root/.cache
WORKDIR /tmp/
COPY --from=builder /go/src/github.com/jhoblitt/rook-block-be-gone/$BIN /bin/$BIN
USER 42:42
CMD ["/bin/rook-block-be-gone"]
