package plugins

import (
	"encoding/json"
	"fmt"
	"main.go/minecraft/protocol/packet"
	"main.go/task"
)

type ShowChat struct {
	DstInterfaces []string `json:"dests"`
	Hint          string   `json:"hint"`
	sends         []func(isJson bool, data string)
}

func (o *ShowChat) New(config []byte) Plugin {
	o.DstInterfaces = make([]string, 0)
	err := json.Unmarshal(config, o)
	if err != nil {
		panic(err)
	}
	return o
}

func (o *ShowChat) Inject(taskIO *task.TaskIO, collaborationContext map[string]Plugin) Plugin {
	o.sends = make([]func(isJson bool, data string), 0)
	for _, dst := range o.DstInterfaces {
		dstInterface := collaborationContext[dst].(StringWriteInterface)
		o.sends = append(o.sends, dstInterface.RegStringSender(o.Hint))
	}

	taskIO.AddPacketTypeCallback(packet.IDText, o.onNewTextPacket)
	return o
}

func (o *ShowChat) onNewTextPacket(p packet.Packet, cbID int) {
	pk := p.(*packet.Text)
	for _, send := range o.sends {
		send(false, fmt.Sprintf("%v (%v): %v\n", pk.SourceName, pk.TextType, pk.Message))
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
