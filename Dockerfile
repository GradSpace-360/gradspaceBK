FROM golang:1.23.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# taking build of the project
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM alpine:latest  

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache tzdata

WORKDIR /root/

#copy build file from previous image
COPY --from=builder /app/app .
COPY --from=builder /app/templates ./templates

CMD ["./app"]