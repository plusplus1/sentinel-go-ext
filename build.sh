#!/bin/bash

WORK_DIR=$(cd `dirname $0`; pwd)
echo $WORK_DIR

build_dashboard() {
    cd $WORK_DIR/dashboard/webui/ && \
        npm install && \
        npm run build:prod

    cd $WORK_DIR && go mod vendor &&  \
        go build -mod=vendor -tags="sonic avx" -o bin/sentinel_dashboard dashboard/cmd/main.go
}


build_dashboard

