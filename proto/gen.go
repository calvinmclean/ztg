package proto

//go:generate protoc --go_out=../gen/proto/dice --go_opt=paths=source_relative --go-grpc_out=../gen/proto/dice --go-grpc_opt=paths=source_relative ./dice.proto
//go:generate protoc --go_out=../gen/proto/factorfight --go_opt=paths=source_relative --go-grpc_out=../gen/proto/factorfight --go-grpc_opt=paths=source_relative ./factorfight.proto
//go:generate protoc --go_out=../gen/proto/game --go_opt=paths=source_relative --go-grpc_out=../gen/proto/game --go-grpc_opt=paths=source_relative ./game.proto
