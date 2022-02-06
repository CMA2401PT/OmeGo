package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type LevelSoundEventHandler struct{}

func (l LevelSoundEventHandler) Handle(p packet.Packet, s *Session) error {
	pk := p.(*packet.LevelSoundEvent)
	if pk.SoundType == packet.SoundEventAttackNoDamage && s.C.GameMode().Visible() {
		s.swingingArm.Store(true)
		defer s.swingingArm.Store(false)
		s.C.PunchAir()
	}
	return nil
}
