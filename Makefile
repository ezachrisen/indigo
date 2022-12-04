test: race_test normal_test benchmark

race_test:
	go test -v -race ./...

normal_test:
	go test -count=1 ./...

benchmark:
	@echo "Benchmarking results saved in ./testdata/benchmarks/after.txt"
	@echo "   Update ./testdata/benchmarks/reference.txt to set a new 'before' point."	
	go test  -bench=. -count 5 ./... | tee ./testdata/benchmarks/after.txt
	benchstat ./testdata/benchmarks/reference.txt ./testdata/benchmarks/after.txt

stats:
	benchstat ./testdata/benchmarks/reference.txt ./testdata/benchmarks/after.txt


# echo "----- Running benchmarks"
# if ! [ -x "$(command -v benchstat)" ]
# then
#     go test -run=XXX -bench=.  ./...
# else
#     echo "benchstat is installed; showing comparison with reference after completion"
#     echo "   Results saved in ./testdata/benchmarks/after.txt"
#     echo "   Update ./testdata/benchmarks/reference.txt to set a new 'before' point."
#     go test  -bench=. -count 5 ./... | tee ./testdata/benchmarks/after.txt
# fi

