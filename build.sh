#!/bin/bash

WORK_DIR=$(cd `dirname $0`; pwd)
echo $WORK_DIR


build_dashboard_webui(){
    cd $WORK_DIR/dashboard/webui/ && npm install && npm run build:prod
}

build_dashboard_server() {
    cd $WORK_DIR && go mod vendor && go build -mod=vendor -tags="sonic avx" \
        -o bin/sentinel_dashboard \
        dashboard/cmd/main.go
}

build_dashboard(){
    if [ ! -d ${WORK_DIR}/dashboard/dist ]; then
        build_dashboard_webui
    fi
    build_dashboard_server
}

case "$1" in
    dashboard)
        build_dashboard
        ;;
esac
