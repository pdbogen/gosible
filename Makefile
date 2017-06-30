.PHONY: testbed

gosible: ${shell find -name \*.go}
	go build -o gosible github.com/pdbogen/gosible

test: .testbed_run gosible
	./gosible --root example-payload
	curl -f 127.0.0.1:10080

testbed: .testbed_run

.testbed_run: .testbed_docker Makefile
	docker rm -f testbed || true
	docker run -p 10022:22 -p 10080:80 --name testbed -d testbed > .testbed_run

.testbed_docker: ${shell find testbed/} Makefile
	docker build -t testbed testbed/
	touch .testbed_docker
