package session

import (
	"time"

	"github.com/google/uuid"
	"main.go/minecraft/protocol/packet"
)

// EmoteHandler handles the Emote packet.
type EmoteHandler struct {
	LastEmote time.Time
}

// Handle ...
func (h *EmoteHandler) Handle(p packet.Packet, s *Session) error {
	pk := p.(*packet.Emote)

	if pk.EntityRuntimeID != selfEntityRuntimeID {
		return ErrSelfRuntimeID
	}
	if time.Since(h.LastEmote) < time.Second {
		return nil
	}
	h.LastEmote = time.Now()
	emote, err := uuid.Parse(pk.EmoteID)
	if err != nil {
		return err
	}
	for _, viewer := range s.c.World().Viewers(s.c.Position()) {
		viewer.ViewEmote(s.c, emote)
	}
	return nil
}
