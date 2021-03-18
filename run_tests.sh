#!/bin/bash
# run_test.sh runs all tests in the module:
#  - normal tests
#  - benchmarks
#  - normal tests with the -race detector turned on
echo "----- Test run takes 3-6 minutes -----"
echo "----- Invalidating test cache"
go clean -testcache ./...

echo "----- Running tests (with -short flag)"
go test -short ./...

echo "----- Running benchmarks"
if ! [ -x "$(command -v benchstat)" ]
then
    go test -run=XXX -bench=. ./...
else
    echo "benchstat is installed; showing comparison with reference after completion"
    echo "   Results saved in ./testdata/benchmarks/after.txt"
    echo "   Update ./testdata/benchmarks/reference.txt to set a new 'before' point."
    go test -run=XXX -bench=. ./...  | tee ./testdata/benchmarks/after.txt
    benchstat ./testdata/benchmarks/reference.txt ./testdata/benchmarks/after.txt
fi

echo "----- Running race condition tests"
go test -run=Concurr -v -race ./...
