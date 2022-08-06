#!/bin/bash

build_dashboard() {
    go mod vendor
    go build -mod=vendor -tags="sonic avx" -o bin/sentinel_dashboard dashboard/cmd/main.go
}


build_dashboard

