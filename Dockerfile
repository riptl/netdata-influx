FROM golang:alpine as builder
RUN apk add git
WORKDIR /app
ADD go.* /app/
RUN go mod download
ADD ./ /app/
RUN CGO_ENABLED=0 go build \
    -installsuffix cgo \
    -ldflags="-s -w" \
    -o /go/bin/ni \
    .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/ni /bin/ni
ENV NI_LOG_TIMESTAMPS=false
CMD ["/bin/ni"]
