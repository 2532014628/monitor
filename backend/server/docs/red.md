# 服务器容器配置使用手册
## 第一步：启动tdengine
docker run -itd \
--name tdengine \
--restart always \
--network root_default \
-e TAOS_FQDN="tdengine" \
-e TAOS_FIRST_EP="tdengine"  \
-e TZ="Asia/Shanghai" \
-e TAOS_SERVER_PORT="6030" \
-p 6041:6041  \
-p 6030:6030 \
ccr.ccs.tencentyun.com/wujinhao/tdengine:3330
## 第二步:启动postgres
docker run -it \
--name postgres \
--restart always \
--network root_default \
--privileged \
-e POSTGRES_USER="postgres" \
-e POSTGRES_PASSWORD=123 \
-p 5432:5432 \
-v /usr/local/software/postgres/data:/var/lib/postgresql/data \
-d postgres

## 第三步:启动redis
docker run \
-itd \
--name redis \
--network root_default \
-p 6379:6379 \
--restart unless-stopped \
-v /usr/local/software/redis/data:/data \
-v /usr/local/software/redis/conf/redis.conf:/etc/redis/redis.conf \
redis:latest /etc/redis/redis.conf
[//]: # (redis-server /etc/redis/redis.conf \)

## 第四步:启动程序运行容器
docker   run   -dit  \
-p 8080:8080  \
-p 20024:22  \
--network root_default  \
--name dev  \
--hostname dev  \
--link tdengine:tdengine  \
-v ./go:/root/go/ \
ccr.ccs.tencentyun.com/qzes/environment:td3330-20241212 && docker exec -it dev service ssh restart

