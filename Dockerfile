FROM golang:1.17
WORKDIR /handshake
COPY . .
RUN go mod download && go mod verify
RUN go build