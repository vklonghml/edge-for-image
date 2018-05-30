
# search face api test
IMAGE_URL=http://49.4.5.210:9090/txry/ffdf1c32-3b03-40aa-b2a1-65a255e6b6b8-1521189474151.jpg
url is: http://127.0.0.1:8086/v1/faceSet//addFace, body is {"imageUrl": "http://49.4.5.210:9090/edge-002/ebc80399-21d3-4a73-8390-178c5b20584b-anyushun.png", "externalImageID": ""}

curl -X PUT http://127.0.0.1:8086/v1/faceSet/43/addFace -d '{"imageUrl": "'"$IMAGE_URL"'", "externalImageID": "123456"}'

while :
do
  date
  curl -X GET http://127.0.0.1:8086/v1/faceSet/43/faceSearch?url=$IMAGE_URL
done
