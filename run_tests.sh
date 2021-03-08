#!/bin/bash
echo "Running all tests"
go test ./...
echo "Running benchmarks"
if ! [ -x "$(command -v benchstat)" ]
then
    echo "benchstat is not installed"
    go test -bench=. ./...
else
    echo "benchstat is installed; showing comparison after completion"
    go test -bench=. ./... > ./testdata/benchmarks/after.txt
    benchstat ./testdata/benchmarks/reference.txt ./testdata/benchmarks/after.txt
fi

echo "Running race condition tests"
go test -race ./...
