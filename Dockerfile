FROM golang:latest
RUN apt-get update && apt-get install -y --no-install-recommends fuse && rm -rf /var/lib/apt/lists/*
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go build -o main .
CMD ["/app/main", "/mnt"]