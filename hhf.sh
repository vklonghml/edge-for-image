
# search face api test
IMAGE_URL=http://49.4.5.210:9090/test_faceset1/8c88608d-a4d8-46e7-8104-b950dc5f3091-pujie.jpg

curl -X PUT http://127.0.0.1:8086/v1/faceSet/25/addFace -d '{"imageUrl": "'"$IMAGE_URL"'", "externalImageID": "123456"}'
