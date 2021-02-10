FROM golang:1.15-alpine as build

ENV CGO_ENABLED=0
ENV GO111MODULE=on

WORKDIR /go/src/github.com/alexellis/derek
COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=${CGO_ENABLED} go test $(go list ./... | grep -v /vendor/) -cover
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=${CGO_ENABLED} go build -mod=vendor -a -installsuffix cgo -o derek .

FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:3.13 as ship

RUN apk --no-cache add ca-certificates

RUN addgroup -S app && adduser -S -g app app
RUN mkdir -p /home/app

WORKDIR /home/app

COPY --from=build /go/src/github.com/alexellis/derek/derek derek

RUN chown -R app /home/app

USER app

ENV VALIDATE_HMAC="true"
ENV VALIDATE_CUSTOMERS="false"
ENV DCO_STATUS_CHECKS="true"

EXPOSE 8080
CMD ["./derek"]
