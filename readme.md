### 说明

本接口只支持 websocket 连接，用于接收任务回调，其它接口请使用 http 回调接口

> https://github.com/botsphp/douyin_crawler

### 编译

```
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build client.go
```

> windows 下编译方法
```
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build client.go
```

### 如何使用

1. 下载 client.exe 和 config.json, 如保存到 `D:\test`
2. 修改 config.json 中的信息，修改 token 和 server 为客服分配的信息
3. 如果本地有 Redis 环境，需要修改 config.json 中的 Redis 配置信息
4. 进入命令行，cd D:\test，然后执行 client.exe，如果连接正常会返回服务器响应的信息，如果有任务执行成功，则会收到数据，并保存到 result.log 文件中
5. 通过其它方式，提交任务，如 `curl -s -m 5 -d '{"token":"36ea7692e261cc32f593b2cd7eb7dc6c","type":"crawler_search_user","search":"面膜","num":20}' \
https://service.yundou.me/`
6. 等待任务执行，执行完成，client.exe 会收到任务结果，将写入 Reids 和 result.log