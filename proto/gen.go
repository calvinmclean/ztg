package proto

//go:generate sh -c "protoc -I . --go_out=../gen/go $(find . -name '*.proto')"
