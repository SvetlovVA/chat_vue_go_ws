module backend

go 1.24.4

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20240311132316-a219d84964c2

require (
	github.com/gorilla/websocket v1.4.1
	github.com/improbable-eng/grpc-web v0.15.0
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cenkalti/backoff/v4 v4.1.1 // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/rs/cors v1.7.0 // indirect
	golang.org/x/net v0.46.1-0.20251013234738-63d1a5100f82 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8 // indirect
	nhooyr.io/websocket v1.8.6 // indirect
)
