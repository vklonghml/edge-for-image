pkill edge-for-image
./edge-for-image \
--static-dir=/opt/edgeServer/images \
--v=2 \
--alsologtostderr \
--db-conn-str="root:@tcp(127.0.0.1:3306)/testaicloud" \
--host=192.168.1.52 \
--public-host=49.4.5.210 \
--port=9090 \
--disk-threshod=80 \
--similarity=92 \
--pic-wait-sec=20 \
--regist-period-sec=5 \
--detect-period-sec=2 \
--regist-cache-size=5 \
--detect-cache-size=5 \
--auto-regist-size=100 \
--aiurl=http://127.0.0.1:8086 >> edge-for-image.log 2>&1 &
