# Installation

```
# Run SSH server
cd ssh-server
docker-compose up

----

# Compile WASM
npm installgit s

GOOS=js GOARCH=wasm go build -o out.wasm wasm.go
go run server.go
# visit localhost:8081
```
