#!/bin/bash

WORK_DIR=$(cd `dirname $0`; pwd)
echo $WORK_DIR

build_dashboard_webui(){
    # Build new frontend (React + Ant Design)
    cd $WORK_DIR/frontend && npm install && npm run build
}

build_dashboard_server() {
    # Compute version from git short tag
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "unknown")
    cd $WORK_DIR && go mod vendor && go build -mod=vendor -tags="sonic avx" \
        -ldflags "-X github.com/plusplus1/sentinel-go-ext/dashboard.Version=$VERSION" \
        -o bin/sentinel_dashboard \
        ./dashboard/cmd/
}

build_dashboard(){
    # Build frontend if dist directory doesn't exist
    if [ ! -d ${WORK_DIR}/frontend/dist ]; then
        build_dashboard_webui
    fi
    build_dashboard_server
}

case "$1" in
    web)
        build_dashboard_webui
        ;;
    srv)
        build_dashboard_server
        ;;
    *)
        build_dashboard
        ;;
esac
