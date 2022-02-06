package server

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sirupsen/logrus"
	"main.go/minecraft"
	"main.go/minecraft/protocol/login"
	"main.go/plugins/world_mirror/server/player/chat"
	"main.go/plugins/world_mirror/server/session"
	"main.go/plugins/world_mirror/server/world"
)

type StartData struct {
	InjectFns *InjectFns
}

type ChatSubscriber struct {
	ChatCb func(a ...interface{})
}

func (s *ChatSubscriber) Message(a ...interface{}) {
	s.ChatCb(a...)
}

type InjectFns struct {
	ChatSubscriber         *ChatSubscriber
	GetNetEaseGameData     func() minecraft.GameData
	GetNetEaseClientData   func() login.ClientData
	GetNetEaseIdentityData func() login.IdentityData
	GetProvider            func() (world.Provider, error)
	GetPos                 func() mgl32.Vec3
	SessionInjectFns       *session.InjectFns
}

type ServerHandle struct {
	ReflectServerConfig Config
	InjectFns           *InjectFns
	Server              *Server
}

func RunServer(h *ServerHandle) {
	log := logrus.New()
	log.Formatter = &logrus.TextFormatter{ForceColors: true}
	log.Level = logrus.DebugLevel

	chat.Global.Subscribe(h.InjectFns.ChatSubscriber)

	srv := New(&h.ReflectServerConfig, log, h.InjectFns)
	srv.CloseOnProgramEnd()
	h.Server = srv

	if err := srv.Start(); err != nil {
		log.Fatalln(err)
	}

	for {
		if _, err := srv.Accept(); err != nil {
			return
		}
	}
}

func StartWithData(sd *StartData) *ServerHandle {
	h := &ServerHandle{
		ReflectServerConfig: DefaultConfig(),
		InjectFns:           sd.InjectFns,
	}
	h.ReflectServerConfig.Server.Name = "Reflect Server"
	h.ReflectServerConfig.Server.AuthEnabled = false
	h.ReflectServerConfig.World.Name = "Reflect Server World"
	h.ReflectServerConfig.World.Folder = "Reflect_Server_World"
	h.ReflectServerConfig.Players.SaveData = false
	go RunServer(h)
	return h
}
