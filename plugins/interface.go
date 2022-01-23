package plugins

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"main.go/define"
	"main.go/task"
	"os"
	"strings"
)

type StringWriteInterface interface {
	RegStringSender(name string) func(isJson bool, data string)
}

type StringReadInterface interface {
	RegStringInterceptor(name string, intercept func(isJson bool, data string) (bool, string)) int
	RemoveStringInterceptor(interceptID int)
}

type stringInterceptor struct {
	name      string
	intercept func(isJson bool, data string) (bool, string)
}
type CliInterface struct {
	Prefix                 string `yaml:"chat_prefix"`
	taskIO                 *task.TaskIO
	collaborationContext   map[string]define.Plugin
	stringSender           map[string]func(isJson bool, data string)
	stringInterceptorCount int
	stringInterceptors     map[int]stringInterceptor
	screenLock             chan int
}

func (u *CliInterface) New(config []byte) define.Plugin {
	err := yaml.Unmarshal(config, u)
	if err != nil {
		panic(err)
	}
	u.stringSender = make(map[string]func(isJson bool, data string))
	u.stringInterceptorCount = 0
	u.stringInterceptors = make(map[int]stringInterceptor)
	//u.screenLock = make(chan int)
	return u
}

func (u *CliInterface) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	u.taskIO = taskIO
	u.collaborationContext = collaborationContext
	return u
}

func (u *CliInterface) RegStringSender(name string) func(isJson bool, data string) {
	_, hasK := u.stringSender[name]
	if hasK {
		return nil
	}
	fn := func(isJson bool, data string) {
		u.NewString(name, isJson, data)
	}
	u.stringSender[name] = fn
	return fn
}

func (u *CliInterface) RegStringInterceptor(name string, intercept func(isJson bool, data string) (bool, string)) int {
	c := u.stringInterceptorCount + 1
	if c == 0 {
		panic("RegStringInterceptors Over Limit!")
	}
	//_,hasK:=u.stringInterceptors[c]
	//for hasK{
	//	c+=1
	//	_,hasK=u.stringInterceptors[c]
	//}
	u.stringInterceptorCount = c
	u.stringInterceptors[c] = stringInterceptor{name: name, intercept: intercept}
	return c
}

func (u *CliInterface) RemoveStringInterceptor(interceptID int) {
	_, ok := u.stringInterceptors[interceptID]
	if ok {
		delete(u.stringInterceptors, interceptID)
	}
}

func (u *CliInterface) RemoveOnCmdFeedBackOnCb(interceptorID int) bool {
	_, ok := u.stringInterceptors[interceptorID]
	if ok {
		delete(u.stringInterceptors, interceptorID)
	}
	return ok
}

func (u *CliInterface) NewString(source string, isJson bool, data string) {
	//<-u.screenLock
	data = strings.TrimSpace(data)
	if isJson {
		var anyData interface{}
		err := json.Unmarshal([]byte(data), &anyData)
		if err == nil {
			fmt.Printf("(%v) Json> %v\n", source, anyData)
		} else {
			fmt.Printf("(%v) BrokenJson(%v)> %v\n", source, err, data)
		}
	} else {
		fmt.Printf("(%v) Text> %v\n", source, data)
	}
}

func (u *CliInterface) REPL() {
	fmt.Printf("")
	reader := bufio.NewReader(os.Stdin)
	s, _ := reader.ReadString('\n')
	s = strings.TrimSpace(s)
	var catch bool
	for _, intercept := range u.stringInterceptors {
		fmt.Printf("(%s) < %s\n", intercept.name, s)
		catch, s = intercept.intercept(false, s)
		if catch {
			break
		}
	}
	if !catch {
		fmt.Printf("(%s) < %s\n", "game", u.Prefix+s)
		u.taskIO.Say(false, u.Prefix+s)
	}
}

func (u *CliInterface) Routine() {
	u.taskIO.WaitInit()
	for {
		u.REPL()
	}
}

func (u *CliInterface) Close() {
}
