package define

import "main.go/task"

type Plugin interface {
	New(config []byte) Plugin
	Inject(taskIO *task.TaskIO, collaborationContext map[string]Plugin) Plugin
	Routine()
	Close()
}

type StringWriteInterface interface {
	RegStringSender(name string) func(isJson bool, data string)
}

type StringReadInterface interface {
	RegStringInterceptor(name string, intercept func(isJson bool, data string) (bool, string)) int
	RemoveStringInterceptor(interceptID int)
}

type InterceptFn func(isJson bool, data string) (bool, string)
