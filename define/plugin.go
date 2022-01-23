package define

import "main.go/task"

type Plugin interface {
	New(config []byte) Plugin
	Inject(taskIO *task.TaskIO, collaborationContext map[string]Plugin) Plugin
	Routine()
	Close()
}
