DUCKDB_VERSION=0.8.1

.PHONY: install
install:
	go install .

.PHONY: examples
examples:
	go run examples/simple.go

.PHONY: test
test:
	go test -v -race -count=1 .

.PHONY: deps.source
deps.source:
	curl -Lo libduckdb.zip https://github.com/duckdb/duckdb/releases/download/v${DUCKDB_VERSION}/libduckdb-src.zip
	unzip -o libduckdb.zip
	rm libduckdb.zip
	echo '#ifdef GODUCKDB_FROM_SOURCE' > duckdb.hpp.tmp; cat duckdb.hpp >> duckdb.hpp.tmp; echo '\n#endif' >> duckdb.hpp.tmp; mv duckdb.hpp.tmp duckdb.hpp
	echo '#ifdef GODUCKDB_FROM_SOURCE' > duckdb.cpp.tmp; cat duckdb.cpp >> duckdb.cpp.tmp; echo '\n#endif' >> duckdb.cpp.tmp; mv duckdb.cpp.tmp duckdb.cpp

.PHONY: deps.darwin.amd64
deps.darwin.amd64:
	if [ "$(shell uname -s | tr '[:upper:]' '[:lower:]')" != "darwin" ]; then echo "Error: must run build on darwin"; false; fi
	g++ -std=c++11 -O3 --target=x86_64-apple-macos11 -DGODUCKDB_FROM_SOURCE -DNDEBUG -c duckdb.cpp
	ar rvs libduckdb.a duckdb.o
	mv libduckdb.a deps/darwin_amd64/libduckdb.a

.PHONY: deps.darwin.arm64
deps.darwin.arm64:
	if [ "$(shell uname -s | tr '[:upper:]' '[:lower:]')" != "darwin" ]; then echo "Error: must run build on darwin"; false; fi
	g++ -std=c++11 -O3 --target=arm64-apple-macos11 -DGODUCKDB_FROM_SOURCE -DNDEBUG -c duckdb.cpp
	ar rvs libduckdb.a duckdb.o
	mv libduckdb.a deps/darwin_arm64/libduckdb.a

.PHONY: deps.linux.amd64
deps.linux.amd64:
	if [ "$(shell uname -s | tr '[:upper:]' '[:lower:]')" != "linux" ]; then echo "Error: must run build on linux"; false; fi
	g++ -std=c++11 -O3 -DGODUCKDB_FROM_SOURCE -DNDEBUG -c duckdb.cpp
	ar rvs libduckdb.a duckdb.o
	mv libduckdb.a deps/linux_amd64/libduckdb.a

.PHONY: deps.linux.arm64
deps.linux.arm64:
	if [ "$(shell uname -s | tr '[:upper:]' '[:lower:]')" != "linux" ]; then echo "Error: must run build on linux"; false; fi
	aarch64-linux-gnu-g++ -std=c++11 -O3 -DGODUCKDB_FROM_SOURCE -DNDEBUG -c duckdb.cpp
	aarch64-linux-gnu-gcc-ar rvs libduckdb.a duckdb.o
	mv libduckdb.a deps/linux_arm64/libduckdb.a
