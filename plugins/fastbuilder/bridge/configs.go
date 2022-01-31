package bridge

import (
	"main.go/minecraft"
	"main.go/task"
)

var GetConn func() *minecraft.Conn
var Operator string
var Supervisors string
var BypassedTaskIO *task.TaskIO
