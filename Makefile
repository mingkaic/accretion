.PHONY: dgraph_up
dgraph_up:
	docker run --name=dgraph -d -p 6080:6080 -p 8080:8080 -p 9080:9080 -p 8000:8000 -v /mnt/dgraph:/dgraph dgraph/standalone:v20.03.0

.PHONY: dgraph_down
dgraph_down:
	docker rm -f dgraph

.PHONY: dgraph_restart
dgraph_restart: dgraph_down dgraph_up

.PHONY: dgraph_clean
dgraph_clean: dgraph_down
	sudo rm -rf /mnt/dgraph

# proto generation
GOOGLE_API := github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis
ONNX_PROTO := proto/internal/onnx/onnx.proto
ANNOTATIONS_PROTO := proto/google/api/annotations.proto

proto/profile.pb.go: ${ONNX_PROTO} ${ANNOTATIONS_PROTO}
	cd proto && buf generate

${ONNX_PROTO}:
	mkdir -p proto/internal/onnx
	cp $$GOPATH/src/github.com/mingkaic/onnx_go/onnx/onnx.proto ${ONNX_PROTO}
	echo 'option go_package = "github.com/mingkaic/onnx_go/onnx";' >> ${ONNX_PROTO}

${ANNOTATIONS_PROTO}:
	mkdir -p proto/google/api
	cp $$GOPATH/src/${GOOGLE_API}/google/api/annotations.proto ${ANNOTATIONS_PROTO}
	cp $$GOPATH/src/${GOOGLE_API}/google/api/http.proto proto/google/api/http.proto

clean:
	rm -rf proto/internal
	rm -rf proto/google
	rm proto/*.go
