#!/usr/bin/env bash
export IP=127.0.0.1
export FACESET="test31"
. pujie.jpg.base64
#. test.txt
#export IMAGE=`base64 face/普杰.png`

upload_one() {
    curl -X POST http://$IP:9090/api/v1/faces/upload \
    -d '{"filename":"pujie.jpg","imageBase64":"'"${IMAGE}"'", "facesetname":"'"${FACESET}"'"}'
}

count=
main(){
    if [ -z $1 ]; then
        count=1
    else
        count=$1
    fi
    echo "the count is $count"

    for ((i=0;i<$count;i++));do
        echo "uploading $i"
        upload_one
        sleep 0.5
    done
}

main $@
