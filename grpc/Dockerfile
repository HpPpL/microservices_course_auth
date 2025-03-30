FROM golang:1.20.3-alpine AS builder

COPY . /github.com/HpPpL/microservices_course_auth/grpc/source/
WORKDIR /github.com/HpPpL/icroservices_course_auth/grpc/source/

RUN go mod download
RUN go build -o ./bin/crud_server cmd/grpc_server/main.go

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /github.com/HpPpL/microservices_course_auth/grpc/source/bin/crud_server .

CMD ["./crud_server"]