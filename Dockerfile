FROM golang:latest as builder

WORKDIR /app

# install deps
COPY go.mod go.sum ./
RUN go mod download

# compile
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .


FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/main .

ADD data data

CMD ["./main"]
