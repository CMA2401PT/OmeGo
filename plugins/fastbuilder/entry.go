package fastbuilder

import (
	"gopkg.in/yaml.v3"
	"main.go/define"
	"main.go/task"
	"strings"
)

type Source struct {
	Use     []string `yaml:"use"`
	RegName string   `yaml:"reg_name"`
	Plugin  string   `yaml:"plugin"`
}

type FastBuilder struct {
	Sources   []Source `yaml:"sources"`
	LogName   string   `yaml:"log_name"`
	LogPlugin string   `yaml:"log_plugin"`
	log       func(isJson bool, data string)
	taskIO    *task.TaskIO
}

func (fb *FastBuilder) New(config []byte) define.Plugin {
	fb.Sources = make([]Source, 0)
	fb.LogName = ""
	fb.LogPlugin = "storage"
	err := yaml.Unmarshal(config, fb)
	if err != nil {
		panic(err)
	}
	return fb
}

func (fb *FastBuilder) Close() {
}

func (fb *FastBuilder) Routine() {

}

func (fb *FastBuilder) onNewText(isJson bool, use []string, data string) (bool, string) {
	data = strings.TrimSpace(data)
	for _, prefix := range use {
		if strings.HasPrefix(data, prefix) {
			//catch
			return true, ""
		}
	}
	// fall through
	return false, data
}

func (fb *FastBuilder) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	fb.taskIO = taskIO
	if fb.LogName != "" {
		fb.log = collaborationContext[fb.LogPlugin].(define.StringWriteInterface).RegStringSender(fb.LogName)
	}
	for _, s := range fb.Sources {
		src := collaborationContext[s.Plugin].(define.StringReadInterface)
		src.RegStringInterceptor(s.RegName, func(isJson bool, data string) (bool, string) {
			return fb.onNewText(isJson, s.Use, data)
		})
	}
	return fb
}
