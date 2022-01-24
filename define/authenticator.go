package define

import (
	"main.go/minecraft"
	"main.go/minecraft/protocol/packet"
)

type Authenticator interface {
	Intercept(conn *minecraft.Conn, pk packet.Packet) (packet.Packet, error)
	GenerateToken() (*minecraft.LoginToken, error)
}
