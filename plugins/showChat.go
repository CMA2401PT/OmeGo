package plugins

import (
	"gopkg.in/yaml.v3"
	"main.go/define"
	"main.go/minecraft/protocol/packet"
	"main.go/task"
	"strconv"
	"strings"
)

type Dst struct {
	Interface string   `yaml:"plugin"`
	Format    string   `yaml:"format"`
	Filter    []string `yaml:"filter"`
}

type ShowChat struct {
	taskIO                 *task.TaskIO
	DstInterfaces          []Dst  `yaml:"dests"`
	Hint                   string `yaml:"hint"`
	sends                  []func(isJson bool, data string)
	stringInterceptorCount int
	stringInterceptors     map[int]stringInterceptor
}

func (o *ShowChat) New(config []byte) define.Plugin {
	o.DstInterfaces = make([]Dst, 0)
	err := yaml.Unmarshal(config, o)
	o.stringInterceptorCount = 0
	o.stringInterceptors = make(map[int]stringInterceptor)
	if err != nil {
		panic(err)
	}
	return o
}

func (o *ShowChat) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	o.sends = make([]func(isJson bool, data string), 0)
	for _, dst := range o.DstInterfaces {
		dstInterface := collaborationContext[dst.Interface].(define.StringWriteInterface)
		o.sends = append(o.sends, dstInterface.RegStringSender(o.Hint))
	}
	o.taskIO = taskIO
	taskIO.AddPacketTypeCallback(packet.IDText, o.onNewTextPacket)
	return o
}

func (o *ShowChat) RegStringInterceptor(name string, intercept func(isJson bool, data string) (bool, string)) int {
	c := o.stringInterceptorCount + 1
	if c == 0 {
		panic("RegStringInterceptors Over Limit!")
	}
	//_,hasK:=u.stringInterceptors[c]
	//for hasK{
	//	c+=1
	//	_,hasK=u.stringInterceptors[c]
	//}
	o.stringInterceptorCount = c
	o.stringInterceptors[c] = stringInterceptor{name: name, intercept: intercept}
	return c
}

func (o *ShowChat) onNewTextPacket(p packet.Packet) {
	pk := p.(*packet.Text)
	r := strings.NewReplacer("[src]", strings.TrimSpace(pk.SourceName), "[msg]", strings.TrimSpace(pk.Message), "[type]", strconv.Itoa(int(pk.TextType)))
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
