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