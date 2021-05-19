FROM golang:1.16-alpine AS builder

# sqlite3 reqruies cgo
ENV CGO_ENABLED=1

# Need gcc for compiling sqlite3
RUN apk add build-base

WORKDIR /build
COPY . .

# Download go modules
RUN go get

# Build statically
RUN go build -ldflags="-extldflags=-static" -tags "sqlite_omit_load_extension sqlite_fts5" -a -o greetsgraf


# Create a new tiny image
FROM scratch
COPY --from=0 --chown=0:0 /build/greetsgraf /
COPY --from=0 --chown=65534:0 /build/static /static
EXPOSE 8000
CMD ["/greetsgraf", "-serve", "-listen", ":8000", "-static", "/static", "-db", "/db/greetsgraf.db"]
