#!/bin/bash
# run_test.sh runs all tests in the module:
echo "----- Running tests with race condition flag"
go test -v -race ./...

echo "----- Running benchmarks"
if ! [ -x "$(command -v benchstat)" ]
then
    go test -run=XXX -bench=. ./...
else
    echo "benchstat is installed; showing comparison with reference after completion"
    echo "   Results saved in ./testdata/benchmarks/after.txt"
    echo "   Update ./testdata/benchmarks/reference.txt to set a new 'before' point."
    go test -run=XXX -bench=.  ./... | tee ./testdata/benchmarks/after.txt
    benchstat ./testdata/benchmarks/reference.txt ./testdata/benchmarks/after.txt
fi

