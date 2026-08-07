# Build Qsim's web build (wasm) and the share-service into a single image
# that serves both the static assets and the share API on one origin.
#
# Build:  docker build -t qsim .          (context = repo root)
# Run:    docker run -p 8080:8080 -v qsim-data:/app/data qsim
FROM golang:1.26-alpine AS wasmbuild
WORKDIR /src
RUN apk add --no-cache make gcc g++
COPY . .
WORKDIR /src/script
RUN chmod +x build-wasm.sh && ./build-wasm.sh

FROM golang:1.26-alpine AS apibuild
WORKDIR /svc
COPY share-service/ /svc/
RUN go mod tidy && go build -o /share-service .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=wasmbuild /src/Raylib-Go-Wasm/index /app/static
COPY --from=apibuild /share-service /app/share-service
ENV PORT=8080 \
    STATIC_DIR=/app/static \
    STORAGE=disk \
    QSIM_SHARES_DIR=/app/data
VOLUME /app/data
EXPOSE 8080
CMD ["/app/share-service"]