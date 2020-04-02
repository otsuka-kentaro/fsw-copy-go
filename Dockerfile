FROM golang:1.14.1 as builder

WORKDIR /app

# build settings
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY main.go .

RUN go build -o app main.go
RUN chmod +x /app


FROM alpine:3.11.5

COPY --from=builder /app/app /app

CMD ["/app"]

