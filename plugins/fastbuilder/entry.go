package fastbuilder

import (
	"gopkg.in/yaml.v3"
	"main.go/plugins/define"
	"main.go/plugins/fastbuilder/bridge"
	"main.go/plugins/fastbuilder/configuration"
	"main.go/plugins/fastbuilder/function"
	I18n "main.go/plugins/fastbuilder/i18n"
	fbtask "main.go/plugins/fastbuilder/task"
	"main.go/plugins/fastbuilder/types"
	"main.go/task"
	"strings"
)

type Source struct {
	Use     []string `yaml:"use"`
	RegName string   `yaml:"reg_name"`
	Plugin  string   `yaml:"plugin"`
}

type FastBuilder struct {
	Sources     []Source `yaml:"sources"`
	LogName     string   `yaml:"log_name"`
	LogPlugin   string   `yaml:"log_plugin"`
	Operator    string   `yaml:"operator"`
	Supervisors string   `yaml:"supervisors"`
	Language    string   `yaml:"language"`
	log         func(isJson bool, data string)
	taskIO      *task.TaskIO
}

func (fb *FastBuilder) New(config []byte) define.Plugin {
	fb.Sources = make([]Source, 0)
	fb.LogName = "fastbuilder"
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
			fb.log(isJson, data)
			function.Process(data)
			return true, ""
		}
	}
	// fall through
	return false, data
}

func setUpBridge(fb *FastBuilder) {
	bridge.GetConn = fb.taskIO.ShieldIO.GetConn
	bridge.Supervisors = fb.Supervisors
	bridge.Operator = fb.Operator
	bridge.BypassedTaskIO = fb.taskIO
	configuration.RespondUser = fb.Operator
	I18n.SetLanguage(fb.Language)
	configuration.UserToken = fb.taskIO.StartConfig.FBMCConfig.FBToken
}

func initBridge() {
	types.ForwardedBrokSender = fbtask.BrokSender
	I18n.Init()
	function.InitInternalFunctions()
	fbtask.InitTaskStatusDisplay()
	//world_provider.Init()
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
	setUpBridge(fb)
	initBridge()
	return fb
}
