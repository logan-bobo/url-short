FROM alpine:3.20.0 AS base

RUN apk update
RUN apk upgrade

FROM base AS builder

RUN apk add --update go=1.22.6-r0

WORKDIR /build

ADD . /build

RUN go build -o main .

FROM base AS tester

RUN apk add --update go=1.22.6-r0

RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.59.1

WORKDIR /opt/url-short/

FROM base AS production

WORKDIR /opt/url-short/

COPY --from=builder /build/main .

CMD ["./main"]
