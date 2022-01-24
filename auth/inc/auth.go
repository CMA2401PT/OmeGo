package inc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"main.go/minecraft"
	"main.go/minecraft/protocol/packet"
)

type Authenticator struct {
	Address string
}

func (authenticator *Authenticator) Intercept(conn *minecraft.Conn, pk packet.Packet) (packet.Packet, error) {
	return pk, nil
}

func (authenticator *Authenticator) GenerateToken() (*minecraft.LoginToken, error) {
	key, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	return &minecraft.LoginToken{
		PrivateKey: key,
		UserToken:  "",
		Address:    authenticator.Address,
		NetWork:    "raknet",
	}, nil
}
