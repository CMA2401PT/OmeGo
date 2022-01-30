package plugins

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"main.go/define"
	"main.go/minecraft/protocol/packet"
	"main.go/task"
	"strconv"
	"strings"
)

type ReadChat struct {
	taskIO                 *task.TaskIO
	Format                 string `json:"format"`
	User                   string `json:"user"`
	stringInterceptorCount int
	stringInterceptors     map[int]stringInterceptor
}

func (o *ReadChat) New(config []byte) define.Plugin {
	err := yaml.Unmarshal(config, o)
	o.stringInterceptorCount = 0
	o.stringInterceptors = make(map[int]stringInterceptor)
	if err != nil {
		panic(err)
	}
	return o
}

func (o *ReadChat) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	o.taskIO = taskIO
	taskIO.AddPacketTypeCallback(packet.IDText, o.onNewTextPacket)
	return o
}

func (o *ReadChat) RegStringInterceptor(name string, intercept func(isJson bool, data string) (bool, string)) func() {
	c := o.stringInterceptorCount + 1
	if c == 0 {
		panic("RegStringInterceptors Over Limit!")
	}
	o.stringInterceptorCount = c
	o.stringInterceptors[c] = stringInterceptor{name: name, intercept: intercept}
	return func() {
		delete(o.stringInterceptors, c)
	}
}

func (o *ReadChat) onNewTextPacket(p packet.Packet) {
	pk := p.(*packet.Text)
	if pk.SourceName != o.User {
		return
	}
	r := strings.NewReplacer("[src]", strings.TrimSpace(pk.SourceName), "[msg]", strings.TrimSpace(pk.Message), "[type]", strconv.Itoa(int(pk.TextType)))
	outStr := r.Replace(o.Format)
	fmt.Println("Chat Interface: ", outStr)
	var catch bool
	for _, intercept := range o.stringInterceptors {
		catch, outStr = intercept.intercept(false, outStr)
		if catch {
			break
		}
	}
}

func (o *ReadChat) Routine() {

}

func (o *ReadChat) Close() {

}
