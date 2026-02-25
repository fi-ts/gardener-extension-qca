FROM golang:1.25 AS builder

WORKDIR /go/src/github.com/fi-ts/gardener-extension-qca
COPY . .
RUN make install \
 && strip /go/bin/gardener-extension-qca

FROM alpine:3.22
WORKDIR /
COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-qca /gardener-extension-qca
CMD ["/gardener-extension-qca"]
