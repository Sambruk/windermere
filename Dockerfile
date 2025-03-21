FROM golang:1-alpine AS base
WORKDIR /build
COPY . .
RUN cd cmd/windermere && go build

FROM alpine
COPY --from=base /build/cmd/windermere/windermere /bin
WORKDIR /workdir
CMD ["/bin/windermere", "config.yaml"]

