# Share service for Qsim.

Standalone Go HTTP server that serves the Qsim web app (static files) and the
share API used to publish circuits as short links:

    POST /api/share         {"data": "<circuit>"}  ->  {"id": "abc123..."}
    GET  /api/share/<id>    ->  {"data": "..."}
    GET  /healthz           ->  ok

Storage is pluggable: a local filesystem store (default, fine for local dev)
or an S3-backed store for AWS (set STORAGE=s3). The server also serves the
compiled web app from STATIC_DIR so one origin hosts everything (no CORS).

## Local dev

    go run .                          # serves ./static on :8080

The static folder is produced by the wasm build script at the repo root
(Raylib-Go-Wasm/index). See deploy/README.md for AWS deployment.

Run the repo's own build first:
    powershell -File script/build-wasm.ps1
    go run .   # from this directory (with STATIC_DIR pointing at the repo)

Endpoints must be reachable by the app; the web build calls the same-origin
path "/api/share".
