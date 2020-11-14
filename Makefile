protofiles = ./testdata/proto/*.proto
pbfiles =  ./testdata/school/*.pb.go


all: $(pbfiles) test
	go test -v ./...

$(pbfiles): $(protofiles)
	protoc -I . testdata/proto/*.proto --go_out=plugins=grpc:testdata/school

clean:
	rm -f $(pbfiles)

.PHONY:	test

test: $(pbfiles)
	go test -v -run ExampleSchool ./... 

