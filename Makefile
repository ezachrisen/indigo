protofiles = ./examples/proto/*.proto
pbfiles =  ./examples/school/*.pb.go


all: $(pbfiles) test
	go test -v ./...

$(pbfiles): $(protofiles)
	protoc -I . examples/proto/*.proto --go_out=plugins=grpc:examples/school

clean:
	rm -f $(pbfiles)

.PHONY:	test

test: $(pbfiles)
	go test -v github.com/ezachrisen/rules/...
