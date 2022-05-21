FROM golang:1.18-alpine3.15 AS builder
WORKDIR /src
COPY . .
RUN go mod download
RUN go build -o app .

FROM alpine:3.14.6
LABEL maintainer="Schmille"
ENV GIN_MODE=release
WORKDIR /program
COPY --from=builder /src/app ./
COPY /templates ./templates
CMD ["./app"]