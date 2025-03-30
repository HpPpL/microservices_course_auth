FROM golang:1.20.3-alpine AS builder

COPY grpc /github.com/HpPpL/microservices_course_auth/grpc/source/grpc
COPY go.mod /github.com/HpPpL/microservices_course_auth/grpc/source/
COPY go.sum /github.com/HpPpL/microservices_course_auth/grpc/source/
WORKDIR /github.com/HpPpL/microservices_course_auth/grpc/source/

RUN go mod download
RUN go build -o ./grpc/bin/crud_server ./grpc/cmd/grpc_server/main.go

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /github.com/HpPpL/microservices_course_auth/grpc/source/grpc/bin/crud_server .

CMD ["./crud_server"]