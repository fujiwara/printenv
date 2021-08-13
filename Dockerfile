FROM golang:1.16.7 AS build

WORKDIR /go/src/printenv
COPY ./ /go/src/printenv
ENV CGO_ENABLED=0
RUN go build -o printenv .

FROM scratch
ENV PORT=8080
COPY --from=build /go/src/printenv/printenv /usr/local/bin/printenv
ENTRYPOINT ["/usr/local/bin/printenv"]
