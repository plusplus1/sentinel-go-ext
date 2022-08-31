
.PHONY: build
build:
	/bin/bash build.sh dashboard

.PHONY: format
format:
	@for s in $(shell git status -s | grep -e '.go$$'|grep -v '^D ' |awk '{print $$NF}'); do \
		echo "go fmt $$s";\
		go fmt $$s;\
	done


