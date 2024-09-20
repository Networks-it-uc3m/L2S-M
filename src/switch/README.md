1. Modify the api/v1/ned.proto 
2. Generate protofiles:
```bash
protoc -I=api/v1 --go_out=paths=source_relative:./pkg/nedpb api/v1/ned.proto
```