
# search face api test
IMAGE_URL=http://49.4.5.210:9090/huawei-1/b0479d17-6b55-4eab-8e02-2347562bc6b1-1523875205207.jpg

curl -X PUT http://127.0.0.1:8086/v1/faceSet/25/addFace -d '{"imageUrl": "'"$IMAGE_URL"'", "externalImageID": "123456"}'

while :
do
  date
  curl -X GET http://127.0.0.1:8086/v1/faceSet/25/faceSearch?url=$IMAGE_URL
done