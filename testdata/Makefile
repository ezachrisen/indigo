protofiles = ./proto/*.proto
outputdir=./school
protodir=./proto
pbfiles =  ./school/*.pb.go



all: $(pbfiles) test
	go test -v ./...

$(pbfiles): $(protofiles)
	protoc --go_out=$(outputdir) --go_opt=paths=source_relative \
	--go-grpc_out=$(outputdir) --go-grpc_opt=paths=source_relative,require_unimplemented_servers=false \
	-I=$(protodir) \
	$(protodir)/*.proto

#protoc -I . $(protofiles) --go_out=plugins=grpc:school

clean:
	rm -f $(pbfiles)

.PHONY:	test

test: $(pbfiles)
	cd .. && go test -v ./... 

