package session

import (
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"time"
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
	for _, viewer := range s.C.World().Viewers(s.C.Position()) {
		viewer.ViewEmote(s.C, emote)
	}
	return nil
}
