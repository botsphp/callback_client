package main

import (
    "bytes"
    "compress/zlib"
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/go-redis/redis/v7"
    "github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{}
var muList sync.Mutex = sync.Mutex{}

//客户端映射链表
var client = map[string]int{}

//消费者链表
var consumer = map[string]int{}

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

func Zip(src []byte) []byte {
    var in bytes.Buffer
    w := zlib.NewWriter(&in)
    w.Write(src)
    w.Close()
    return in.Bytes()
}

func queue_push(token string, result []byte) {
    zip := Zip(result)
    Redis.LPush(token, zip)
}

func queue_pop(token string, c *websocket.Conn) {
    for {
        msg, err := Redis.BRPop(time.Second*10, token).Result()
        if err != nil {
            muList.Lock()
            consumer[token] -= 1
            muList.Unlock()

            return
        }

        if mt, ok := client[token]; ok {
            err = c.WriteMessage(mt, []byte(msg[1]))

            //写入失败，可能是连接断开
            //考虑是否做重连处理
            if err != nil {
                log.Println("write:", err)

                muList.Lock()
                consumer[token] -= 1
                delete(client, token)
                muList.Unlock()

                //放回队列等待消费
                Redis.LPush(token, msg[1])
                return
            }
        }
    }
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

            muList.Lock()
            if consumer[token.Token] < 3 {
                consumer[token.Token] += 1

                //最多开启3个消费进程
                go queue_pop(token.Token, c)
            }
            muList.Unlock()

            log.Printf("客户端: %s, 共有连接: %d", token, len(client))
        }

        err = c.WriteMessage(mt, message)
        if err != nil {
            log.Println("write:", err)
            break
        }
    }
}

func home(w http.ResponseWriter, r *http.Request) {
    token := strings.Trim(r.URL.Path, "/")

    //回调是 json 时，是任务回调
    if r.Method == "POST" {
        if len(token) != 32 {
            w.Write([]byte(`{"code":400, "msg":"token is missing"}`))
            return
        }

        result, err := ioutil.ReadAll(r.Body)
        if err != nil {
            log.Println("error:", err)
            w.Write([]byte(`{"code":200, "msg":"body is not json data"}`))
            return
        }

        //队列中最多保留 1000 条消息
        count, err := Redis.LLen(token).Result()
        if err != nil {
            w.Write([]byte(`{"code":200, "msg":"redis has gone away"}`))
            return
        }

        if count < 1000 {
            go queue_push(token, result)
        }

        msg := fmt.Sprintf(`{"code":200, "msg":"success", "queue":"%s"}`, strconv.FormatInt(count, 16))
        w.Write([]byte(msg))
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
