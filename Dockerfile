FROM alpine:3.21 AS base

ARG GO_VER="1.24.2-r0"
ARG GO_CI_VER="2.1.2"

RUN apk update
RUN apk upgrade

FROM base AS builder

RUN apk add --no-cache \
    --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community \
    go=${GO_VER}

WORKDIR /build

ADD . /build

RUN go build -o main .

FROM base AS tester

RUN apk add --update go=${GO_VER}

RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v${GO_CI_VER}

WORKDIR /opt/url-short/

FROM base AS production

WORKDIR /opt/url-short/

COPY --from=builder /build/main .

CMD ["./main"]
