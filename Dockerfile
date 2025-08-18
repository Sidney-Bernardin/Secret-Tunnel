FROM golang:1.24.1

WORKDIR /secret-tunnel

COPY *.mod .
RUN go mod download

COPY . .
RUN go build -o app .
RUN chmod +x entrypoint.sh

ENTRYPOINT ["./entrypoint.sh", "./src"]
