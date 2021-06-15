protofiles = ./proto/*.proto
outputdir=./school
protodir=./proto
pbfiles =  ./school/*.pb.go
descriptor = ../examples/dynamic_schema/school.descriptor



all: $(pbfiles) $(descriptor) test
	go test -v ./...

$(pbfiles) $(descriptor): $(protofiles)
	protoc --go_out=$(outputdir) --go_opt=paths=source_relative \
	--go-grpc_out=$(outputdir) --go-grpc_opt=paths=source_relative,require_unimplemented_servers=false \
	-I=$(protodir) \
	--descriptor_set_out $(descriptor) \
        --include_imports \
        $(protodir)/*.proto


clean:
	rm -f $(pbfiles)

.PHONY:	test docs

test: $(pbfiles)
	cd .. && go test -v ./... 


docs:
	cd .. && goreadme > Usage.md
	cd ../cel && goreadme > README.md
