package plugins

import (
	"encoding/json"
	"main.go/define"
	"main.go/minecraft/protocol/packet"
	"main.go/task"
	"strings"
)

type Dst struct {
	Interface string   `json:"plugin"`
	Format    string   `json:"format"`
	Filter    []string `json:"filter"`
}

type ShowChat struct {
	taskIO        *task.TaskIO
	DstInterfaces []Dst  `json:"dests"`
	Hint          string `json:"hint"`
	sends         []func(isJson bool, data string)
}

func (o *ShowChat) New(config []byte) define.Plugin {
	o.DstInterfaces = make([]Dst, 0)
	err := json.Unmarshal(config, o)
	if err != nil {
		panic(err)
	}
	return o
}

func (o *ShowChat) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	o.sends = make([]func(isJson bool, data string), 0)
	for _, dst := range o.DstInterfaces {
		dstInterface := collaborationContext[dst.Interface].(StringWriteInterface)
		o.sends = append(o.sends, dstInterface.RegStringSender(o.Hint))
	}
	o.taskIO = taskIO
	taskIO.AddPacketTypeCallback(packet.IDText, o.onNewTextPacket)
	return o
}

func (o *ShowChat) onNewTextPacket(p packet.Packet, cbID int) {
	pk := p.(*packet.Text)
	r := strings.NewReplacer("[src]", pk.SourceName, "[msg]", pk.Message, "[type]", string(pk.TextType))
	for i, send := range o.sends {
		filter := o.DstInterfaces[i].Filter
		flag := true
		if filter != nil {
			for _, f := range filter {
				switch f {
				case "not me":
					if o.taskIO.IdentityData().DisplayName == pk.SourceName {
						flag = false
					}
					break
				case "chat only":
					if pk.TextType != packet.TextTypeChat {
						flag = false
					}
					break
				default:
					if strings.Contains(pk.Message, f) {
						flag = false
					}
				}
			}
			if !flag {
				break
			}
		}
		if flag {
			format := o.DstInterfaces[i].Format
			send(false, r.Replace(format))
		}
	}
}

func (o *ShowChat) Routine() {

}

func (o *ShowChat) Close() {

}

//func (u CliInterface) Inject(taskIO *task.TaskIO, collaborationContext map[string]Plugin) Plugin {
//	u.taskIO = taskIO
//	u.collaborationContext = collaborationContext
//	return &u
//}
//
//func (u CliInterface) RegStringSender(name string) func(isJson bool, data string) {
//	_, hasK := u.stringSender[name]
//	if hasK {
//		return nil
//	}
//	fn := func(isJson bool, data string) {
//		u.NewString(name, isJson, data)
//	}
//	u.stringSender[name] = fn
//	return fn
//}
