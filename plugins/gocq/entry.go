package cqchat

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"main.go/plugins/define"
	"main.go/task"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Group struct {
	GID   int64  `yaml:"id"`
	GName string `yaml:"name"`
}

type ChatSettings struct {
	Address           string `yaml:"address"`
	Groups            []Group
	GameMessageFormat string `yaml:"format"`
}

type GoCQ struct {
	taskIO       *task.TaskIO
	settings     *ChatSettings
	upgrader     *websocket.Upgrader
	conn         *websocket.Conn
	connectLock  chan int
	initLock     chan int
	inited       bool
	firstInit    bool
	sendChan     chan string
	stringSender map[string]func(isJson bool, data string)
}

type MetaPost struct {
	Time          int64  `json:"time"`
	PostType      string `json:"post_type"`
	SelfID        int    `json:"self_id"`
	MetaEventType string `json:"meta_event_type"`
}

func ParseMetaPost(data []byte) (MetaPost, error) {
	post := MetaPost{}
	err := json.Unmarshal(data, &post)
	return post, err
}

// receiveRoutine 接收并处理协议端的消息 from QQ
func (cq *GoCQ) receiveRoutine() {
	fmt.Println("CQ-CHAT: Receive Routine Start")
	for {
		// todo 优化
		// msgType为0时 消息正常接收 其他未知
		_, data, err := cq.conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			cq.conn.Close()
			// 如果发送协程还没有尝试重连，那么由发送线程尝试重连
			if cq.inited {
				cq.connect()
			} else {
				<-cq.connectLock
			}
		}
		// 先解析出事件种类(event或message)
		post, err := ParseMetaPost(data)
		if post.PostType == "meta_event" && post.MetaEventType == "lifecycle" {
			fmt.Println("CQ-CHAT: 已成功连接: " + strconv.Itoa(post.SelfID))
			if !cq.inited {
				cq.inited = true
				close(cq.initLock)
			}

		}
		if post.PostType == "message" && err == nil {
			action, err := GetMessageData(cq.settings, data)
			if err != nil || action == nil {
				continue
			}
			// 因为应用场景的原因，我们只关心群应用中的文字消息
			if gmsg, succ := action.(GroupMessage); succ {
				source := gmsg.GetSource()
				user := gmsg.PrivateMessage.Sender.Nickname
				msg := gmsg.UniversalMessage.Message
				//if gmsg.PrivateMessage.MetaPost.PostType == "message" {
				r := strings.NewReplacer("[src]", source, "[msg]", msg, "[user]", user)
				fmsg := r.Replace(cq.settings.GameMessageFormat)
				cq.taskIO.Say(false, fmsg)
			}
		}
		continue
	}
}

// SendMessage
func (cq *GoCQ) sendRoutine() {
	<-cq.initLock
	lastSend := ""
	for {
		for lastSend == "" {
			lastSend = <-cq.sendChan
		}
		echo, _ := uuid.NewUUID()
		for _, g := range cq.settings.Groups {
			qmsg := QMessage{
				Action: "send_group_msg",
				Params: struct {
					GroupID int64  `json:"group_id"`
					Message string `json:"message"`
				}{
					GroupID: g.GID,
					Message: lastSend,
				},
				Echo: echo.String(),
			}
			data, _ := json.Marshal(qmsg)
			err := cq.conn.WriteMessage(1, data)
			if err != nil {
				cq.conn.Close()
				// 如果接收协程还没有尝试重连，那么由发送线程尝试重连
				if cq.inited {
					cq.connect()
				}
				<-cq.initLock
			} else {
				lastSend = <-cq.sendChan
			}
		}
	}
}

func (cq *GoCQ) NewString(srcPlugin string, isJson bool, msg string) {
	cq.sendChan <- msg
}

func (cq *GoCQ) connect() {
	for {
		cq.inited = false
		cq.initLock = make(chan int)
		cq.connectLock = make(chan int)
		u := url.URL{Scheme: "ws", Host: cq.settings.Address}
		var err error
		cq.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			if cq.firstInit {
				panic(err)
			} else {
				log.Println("Go-CQ: CONNECTION ERROR:", err)
			}
		} else {
			close(cq.connectLock)
			break
		}
	}
	if cq.firstInit {
		go cq.receiveRoutine()
		go cq.sendRoutine()
	}
	cq.firstInit = false
}

func (cq *GoCQ) New(config []byte) define.Plugin {
	cq.settings = &ChatSettings{}
	cq.sendChan = make(chan string)
	cq.firstInit = true
	cq.stringSender = make(map[string]func(isJson bool, data string))
	err := yaml.Unmarshal(config, cq.settings)
	if err != nil {
		panic(err)
	}
	cq.upgrader = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	cq.connect()
	<-cq.initLock
	return cq
}

func (cq *GoCQ) Close() {
	cq.conn.Close()
}

func (cq *GoCQ) Routine() {

}

func (cq *GoCQ) RegStringSender(name string) func(isJson bool, data string) {
	_, hasK := cq.stringSender[name]
	if hasK {
		return nil
	}
	fn := func(isJson bool, data string) {
		cq.NewString(name, isJson, data)
	}
	cq.stringSender[name] = fn
	return fn
}

func (cq *GoCQ) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	cq.taskIO = taskIO
	return cq
}
