.PHONY: build frontend backend clean format

# Default target: build both frontend and backend
build: frontend backend

# Build only frontend (React + Ant Design)
frontend:
	/bin/bash build.sh web

# Build only backend (Go server)
backend:
	/bin/bash build.sh srv

# Clean built artifacts
clean:
	rm -rf bin/sentinel_dashboard
	rm -rf frontend/dist

# Format Go source files (only changed files)
format:
	@for s in $(shell git status -s | grep -e '.go$$'|grep -v '^D ' |awk '{print $$NF}'); do \
		echo "go fmt $$s";\
		go fmt $$s;\
	done
