package shield

import (
	"bytes"
	"fmt"
	"main.go/minecraft/alter"
	"math"
	"time"

	"github.com/fatih/color"

	"main.go/minecraft"
	"main.go/minecraft/protocol/packet"
)

func sendInitPackets(conn *minecraft.Conn) error {
	var err error = nil
	runtimeid := fmt.Sprintf("%d", conn.GameData().EntityUniqueID)
	initPacketsContent := [...][]byte{
		[]byte{0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x93, 0xc4, 0xc, 0x53, 0x79, 0x6e, 0x63, 0x55, 0x73, 0x69, 0x6e, 0x67, 0x4d, 0x6f, 0x64, 0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x91, 0x90, 0xc0},
		[]byte{0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x93, 0xc4, 0xf, 0x53, 0x79, 0x6e, 0x63, 0x56, 0x69, 0x70, 0x53, 0x6b, 0x69, 0x6e, 0x55, 0x75, 0x69, 0x64, 0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x91, 0xc0, 0xc0},
		[]byte{0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x93, 0xc4, 0x1f, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x4c, 0x6f, 0x61, 0x64, 0x41, 0x64, 0x64, 0x6f, 0x6e, 0x73, 0x46, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x65, 0x64, 0x46, 0x72, 0x6f, 0x6d, 0x47, 0x61, 0x63, 0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x90, 0xc0},
		bytes.Join([][]byte{[]byte{0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x93, 0xc4, 0xb, 0x4d, 0x6f, 0x64, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x43, 0x32, 0x53, 0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x94, 0xc4, 0x9, 0x4d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0xc4, 0x6, 0x70, 0x72, 0x65, 0x73, 0x65, 0x74, 0xc4, 0x12, 0x47, 0x65, 0x74, 0x4c, 0x6f, 0x61, 0x64, 0x65, 0x64, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x81, 0xc4, 0x8, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49, 0x64, 0xc4},
			[]byte{byte(len(runtimeid))},
			[]byte(runtimeid),
			[]byte{0xc0},
		}, []byte{}),
		[]byte{0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x93, 0xc4, 0x19, 0x61, 0x72, 0x65, 0x6e, 0x61, 0x47, 0x61, 0x6d, 0x65, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x46, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x4c, 0x6f, 0x61, 0x64, 0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x90, 0xc0},
		bytes.Join([][]byte{[]byte{0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x93, 0xc4, 0xb, 0x4d, 0x6f, 0x64, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x43, 0x32, 0x53, 0x82, 0xc4, 0x8, 0x5f, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f, 0x5f, 0xc4, 0x5, 0x74, 0x75, 0x70, 0x6c, 0x65, 0xc4, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x94, 0xc4, 0x9, 0x4d, 0x69, 0x6e, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0xc4, 0xe, 0x76, 0x69, 0x70, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x79, 0x73, 0x74, 0x65, 0x6d, 0xc4, 0xc, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x55, 0x69, 0x49, 0x6e, 0x69, 0x74, 0xc4},
			[]byte{byte(len(runtimeid))},
			[]byte(runtimeid),
			[]byte{0xc0},
		}, []byte{}),
	}
	for pi, content := range initPacketsContent {
		err := conn.WritePacket(&packet.PyRpc{Content: content})
		if err != nil {
			fmt.Printf("MC Session: fail to send the %vth initLock packet, (content: %v)\n", pi, content)
			return fmt.Errorf("fail to send the %vth initLock packet, (content: %v)", pi, content)
		}
	}
	return err
}

type SessionStatus struct {
	connClosed    bool
	connCloseLock chan int
}

func handleMCSession(conn *minecraft.Conn, io *ShieldIO, interceptor PacketInterceptor) error {
	sessionStatus := &SessionStatus{connClosed: false, connCloseLock: make(chan int)}
	closeSession := func() {
		if !sessionStatus.connClosed {
			sessionStatus.connClosed = true
			conn.Close()
			close(sessionStatus.connCloseLock)
		}
	}
	defer closeSession()
	err := sendInitPackets(conn)
	if err != nil {
		return err
	}
	go func() {
		fmt.Println("MC Session: Write Shield Routine Created")
		for {
			if len(io.currentlyWritingGroup) == io.currentlyWritingPacketIndex {
				select {
				case io.currentlyWritingGroup = <-io.packetGroupWriteChan:
					io.currentlyWritingPacketIndex = 0
				case <-sessionStatus.connCloseLock:
					fmt.Println("MC Session (Write Shield Routine): closed from other routine")
					return
				}
			}
			err := conn.WritePacket(io.currentlyWritingGroup[io.currentlyWritingPacketIndex])
			if err != nil {
				fmt.Printf("MC Session (Write Shield Routine): An error (%v) occour when write, anyway, I'll close the connection\n", err)
				closeSession()
				return
			} else {
				io.currentlyWritingPacketIndex++
			}
		}
	}()
	go func() {
		fmt.Println("MC Session: Data Request Shield Routine Created")
		for !sessionStatus.connClosed {
			select {
			case requestID := <-io.connDataRequestFlag:
				switch requestID {
				case 0:
					io.connDataResponseChans.GameDataChain <- conn.GameData()
					break
				case 1:
					io.connDataResponseChans.ClientDataChain <- conn.ClientData()
					break
				}
			case <-sessionStatus.connCloseLock:
				fmt.Println("MC Session (Request Shield Routine): closed from other routine")
				return
			}
		}
	}()
	fmt.Println("MC Session: Read Shield Routine Created")
	for {
		generalPacket, err := conn.ReadPacket()
		if err != nil {
			fmt.Printf("MC Session: An error (%v) occour when read from MC, connection will be closed\n", err)
			closeSession()
			return err
		}
		allowedPacket, err := interceptor(conn, generalPacket)
		if err != nil {
			fmt.Printf("MC Session: An error (%v) occour when intercept packet readed from MC, connection will be closed\n", err)
			closeSession()
			return err
		}
		if allowedPacket == nil {
			continue
		}
		for _, cb := range io.newPacketCallbacks {
			cb(allowedPacket)
		}
	}
}

type PacketInterceptor func(*minecraft.Conn, packet.Packet) (packet.Packet, error)

func (shield *Shield) Routine() {
	color.Blue("MC Routine: Start")
	firstInit := true
	firstConnect := true
	var err error
	for {
		if !firstConnect {
			if !shield.Respawn {
				panic(fmt.Errorf(color.New(color.FgRed).Sprintf("MC Routine: Connection to MC Crashed, and 'Respawn' is false. err=(%v)", err)))
			}
			color.Yellow(fmt.Sprintf("MC Routine: Retrying (%v/%v) connecting to MC...", shield.RetryTimes, shield.MaxRetryTimes))
			shield.RetryTimes += 1
			if shield.MaxRetryTimes != 0 && shield.RetryTimes > shield.MaxRetryTimes {
				panic(fmt.Sprintf("MC Routine: fail to connect to MC after retry (%v)", shield.MaxRetryTimes))
			}
			delayTime := time.Duration(math.Pow(2, float64(shield.RetryTimes-1))) * shield.DelayFactor
			if delayTime > shield.MaxDelay {
				delayTime = shield.MaxDelay
			}
			time.Sleep(delayTime)

		} else {
			color.Blue("MC Routine: Trying connecting to MC...")
			firstConnect = false
		}
		var LoginToken *minecraft.LoginToken
		LoginToken, err = shield.LoginTokenGenerator()
		if err != nil {
			color.Yellow(fmt.Sprintf("MC Routine: fail to get netease login config from fb server (%v)", err))
			continue
		} else {
			color.Green("MC Routine: Get login config success")
		}
		if firstInit {
			for _, cb := range shield.IO.beforeInitCallBacks {
				cb()
			}
		} else {
			for _, cb := range shield.IO.beforeReInitCallBacks {
				cb()
			}
		}

		MCDialer := minecraft.Dialer{}
		if shield.Variant == alter.Variant_Inc {
			MCDialer.ClientData = shield.LoginClientData
			MCDialer.IdentityData = shield.LoginIdentityData
		}
		var conn *minecraft.Conn
		conn, err = MCDialer.Dial(LoginToken)
		if err != nil {
			color.Yellow(fmt.Sprintf("MC Routine: Fail to dial MC (%v)", err))
			continue
		} else {
			color.Green("MC Routine: Connect to MC successfully!")
		}

		if firstInit {
			for _, cb := range shield.IO.initCallBacks {
				cb(conn)
			}
			firstInit = false
		} else {
			for _, cb := range shield.IO.reInitCallBacks {
				cb(conn)
			}
		}
		shield.RespawnTimes += 1
		shield.RetryTimes = 0

		color.Green("MC Routine: Begin MC Session")
		// intercept := WrapPyRPCIntercept(client)
		err = handleMCSession(conn, shield.IO, shield.PacketInterceptor)
		color.Yellow(fmt.Sprintf("MC Routine: An error occour when handle MC Connection (%v)", err))
		for _, cb := range shield.IO.sessionTerminateCallBacks {
			cb()
		}
	}
}
