test:
	go test -count=1 ./...

full_test: race_test test benchmark

race_test:
	go test -v -race ./...


benchmark:
	@COMMIT_HASH=$$(git rev-parse --short HEAD); \
	TIMESTAMP=$$(date +%Y-%m-%d-%H-%M-%S); \
	echo "Benchmarking results saved in ./testdata/benchmarks/$$TIMESTAMP-$$COMMIT_HASH.txt"; \
	echo "This could take a while..."; \
	go test -timeout 30m -bench=. -count 5 -benchmem ./... | tee ./testdata/benchmarks/$$TIMESTAMP-$$COMMIT_HASH.txt

# stats:
# 	@FILES=$$(ls -1 ./testdata/benchmarks/????-??-??-??-??-??-*.txt 2>/dev/null | sort -r | head -2); \
# 	if [ $$(echo "$$FILES" | wc -l) -lt 2 ]; then \
# 		echo "Need at least 2 benchmark files to compare"; \
# 		exit 1; \
# 	fi; \
# 	LATEST=$$(echo "$$FILES" | head -1); \
# 	PREVIOUS=$$(echo "$$FILES" | tail -1); \
# 	echo "Comparing $$PREVIOUS (older) vs $$LATEST (newer)"; \
# 	benchstat "$$PREVIOUS" "$$LATEST"

statsversion:
	@LATEST=$$(ls -1 ./testdata/benchmarks/????-??-??-??-??-??-*.txt 2>/dev/null | sort -r | head -1); \
	VERSIONED=$$(ls -1 ./testdata/benchmarks/v*.txt 2>/dev/null | sort -V | tail -1); \
	if [ -z "$$LATEST" ]; then \
		echo "No dated benchmark files found"; \
		exit 1; \
	fi; \
	if [ -z "$$VERSIONED" ]; then \
		echo "No versioned benchmark files found"; \
		exit 1; \
	fi; \
	echo "Comparing $$VERSIONED (baseline) vs $$LATEST (current)"; \
	benchstat "$$VERSIONED" "$$LATEST" 



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

