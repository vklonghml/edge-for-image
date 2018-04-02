#!/bin/bash

set -e

APP_NAME="edge-for-image"
IMAGE_NAME="edge-for-image"
IMAGE_TAG="latest"

PROJECT_DIR=$(cd "`dirname $0`/../";pwd)

echo "gopath:" $PROJECT_DIR

LOG() 
{
    content=$1
    echo "[`date '+%Y-%m-%d %H:%M:%S'`] $content"
}

if [ -z "$GOPATH" ]; then
    GOPATH=$(cd "$PROJECT_DIR/../../"; pwd)
    echo "GOPATH:" ${GOPATH}
fi

export GOPATH=$GOPATH

BUILD_DIR=$PROJECT_DIR/BUILD_DIR

compile()
{
    LOG "compile $IMAGE_NAME"
    CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags="-s" -o $BUILD_DIR/bin/$IMAGE_NAME $APP_NAME/cmd
    retCode=$?
    if [ $retCode -ne 0 ]; then
        exit $retCode
    fi
}

build_image()
{
    LOG "build $APP_NAME image : $IMAGE_NAME:$IMAGE_TAG"
    image_id=`docker images | grep "$IMAGE_NAME" | grep -w "$IMAGE_TAG" | awk '{print $3}'`
    echo $image_id
    if [ "${image_id}" != "" ]; then   
        images_ids=$(docker ps -a | grep "${IMAGE_NAME}" | awk '{print $1}')
        for i in "${images_ids[@]}"; do
            echo "docker: " ${i}
            if [ "${i}" != "" ]; then
                docker rm -f ${i}
            fi
        done
    fi
    docker images | grep "$IMAGE_NAME" | grep -w "$IMAGE_TAG" | awk '{print $3}' | xargs -r docker rmi -f
    LOG "docker build -t $IMAGE_NAME:$IMAGE_TAG $BUILD_DIR"
    docker build -t $IMAGE_NAME:$IMAGE_TAG $BUILD_DIR
}

run_container()
{
}

curl_to_server()
{
    curl -v -X POST -H 'Content-Type:multipart/form-data' -F uploadfile=@upload.gtpl http://162.3.140.21:9090/upload
}

compile
build_image
run_container

