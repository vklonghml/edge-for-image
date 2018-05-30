
# search face api test
IMAGE_URL=http://49.4.5.210:9090/txry/ffdf1c32-3b03-40aa-b2a1-65a255e6b6b8-1521189474151.jpg

curl -X PUT http://127.0.0.1:8086/v1/faceSet/44/addFace -d '{"imageUrl": "'"$IMAGE_URL"'", "externalImageID": "123456"}'

while :
do
  date
  curl -X GET http://127.0.0.1:8086/v1/faceSet/44/faceSearch?url=$IMAGE_URL
done
