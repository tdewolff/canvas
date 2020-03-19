build:
	GOARCH=wasm GOOS=js go build -o lib.wasm .
	cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" .

serve:
	go run serve.go
