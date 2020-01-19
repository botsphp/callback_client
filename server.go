package main

import (
    "encoding/json"
    "flag"
    "io/ioutil"
    "log"
    "net/http"
    "strings"
    "sync"

    "github.com/go-redis/redis/v7"
    "github.com/gorilla/websocket"
    "github.com/botsphp/callback_client/zip"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{}
var muList sync.Mutex = sync.Mutex{}
var client = map[string]int{}
var Redis = redisInit("127.0.0.1:6379")

type TokenMsg struct {
    Token string `token:`
}

func redisInit(addr string) *redis.Client {
    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: "", // no password set
        DB:       1,  // use default DB
    })

    _, err := client.Ping().Result()

    if err != nil {
        log.Fatal("redis 连接出错:" + err.Error())
    }

    return client
}

func queue_push(token string, result []byte) {

    b := zip.Zip(result)
    log.Println(b)

    Redis.LPush(token, string(result))
}

func queue_pop(callback *websocket.Conn) {

}

func token(w http.ResponseWriter, r *http.Request) {
    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Print("upgrade:", err)
        return
    }
    defer c.Close()

    for {
        mt, message, err := c.ReadMessage()
        if err != nil {
            log.Println("read:", err)
            break
        }
        log.Printf("recv: %s", message)

        var token TokenMsg
        err = json.Unmarshal(message, &token)
        //是任务数据写入队列
        if err == nil && len(token.Token) == 32 {
            muList.Lock()
            client[token.Token] = mt
            muList.Unlock()

            log.Printf("客户端: %s, 共有连接: %d", token, len(client))
        }

        err = c.WriteMessage(mt, message)
        if err != nil {
            log.Println("write:", err)
            break
        }
    }

    go queue_pop(c)
}

func home(w http.ResponseWriter, r *http.Request) {
    path := strings.Trim(r.URL.Path, "/")

    //回调是 json 时，是任务回调
    if r.Method == "POST" {
        if len(path) != 32 {
            w.Write([]byte(`{"code":400, "msg":"token is missing"}`))
            return
        }

        result, err := ioutil.ReadAll(r.Body)
        if err != nil {
            log.Println("error:", err)
            w.Write([]byte(`{"code":200, "msg":"body is not json data"}`))
            return
        }

        if _, ok := client[path]; ok {
            go queue_push(path, result)
        }

        w.Write([]byte(`{"code":200, "serv":"gows"}`))
    } else {
        var hello = `<html><head><meta charset="utf-8"><title>云豆接口 : WebSocket</title></head><body>本接口只支持 websocket 连接，用于接收任务回调</body></html>`
        w.Write([]byte(hello))
    }
}

func main() {
    flag.Parse()
    log.SetFlags(0)
    http.HandleFunc("/token", token)
    http.HandleFunc("/", home)
    log.Fatal(http.ListenAndServe(*addr, nil))
}
