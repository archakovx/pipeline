# build binary
FROM golang:1.13.1-alpine AS build

ARG GOOS
ENV CGO_ENABLED=0 \
    GOOS=$GOOS \
    GOARCH=amd64 \
    CGO_CPPFLAGS="-I/usr/include" \
    UID=0 GID=0 \
    CGO_CFLAGS="-I/usr/include" \
    CGO_LDFLAGS="-L/usr/lib -lpthread -lrt -lstdc++ -lm -lc -lgcc -lz " \
    PKG_CONFIG_PATH="/usr/lib/pkgconfig"

RUN apk add --no-cache git make
RUN go get -u golang.org/x/lint/golint
RUN go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

ARG APP_PKG_NAME
WORKDIR /go/src/$APP_PKG_NAME
COPY ./cmd ./cmd
COPY ./pkg ./pkg
COPY ./vendor ./vendor

ARG VERSION=dev
ARG BINARY_NAME
RUN go vet ./...
RUN golangci-lint run -E gofmt -E golint -E vet
RUN out=$(go fmt ./...) && if [[ -n "$out" ]]; then echo "$out"; exit 1; fi
RUN go test ./...
RUN go build -v \
    -o /out/service \
    -ldflags "-extldflags "-static" -X main.serviceVersion=$VERSION" \
    ./cmd/$BINARY_NAME

# copy to alpine image
FROM alpine:3.8
WORKDIR /app
COPY --from=build /out/service /app/service
CMD ["/app/service"]
