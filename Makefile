generate-operator:
	go run codegen/operator/cleanup/main.go
	go run codegen/operator/main.go

generate-client:
	go run codegen/client/cleanup/main.go
	go run codegen/client/main.go

generate: generate-client generate-operator