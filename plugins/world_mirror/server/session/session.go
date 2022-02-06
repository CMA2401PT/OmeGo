package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/text"
	"go.uber.org/atomic"
	"main.go/plugins/world_mirror/server/block/cube"
	"main.go/plugins/world_mirror/server/internal"
	"main.go/plugins/world_mirror/server/item/inventory"
	"main.go/plugins/world_mirror/server/player/chat"
	"main.go/plugins/world_mirror/server/world"
)

// Session handles incoming packets from connections and sends outgoing packets by providing a thin layer
// of abstraction over direct packets. A Session basically 'controls' an entity.
type Session struct {
	log internal.Logger

	c        Controllable
	Conn     Conn
	handlers map[uint32]PacketHandler

	// onStop is called when the session is stopped. The controllable passed is the controllable that the
	// session controls.
	onStop func(controllable Controllable)

	scoreboardObj atomic.String

	chunkBuf                    *bytes.Buffer
	chunkLoader                 *world.Loader
	chunkRadius, maxChunkRadius int32

	teleportMu  sync.Mutex
	teleportPos *mgl64.Vec3

	entityMutex sync.RWMutex
	// currentEntityRuntimeID holds the runtime ID assigned to the last entity. It is incremented for every
	// entity spawned to the session.
	currentEntityRuntimeID uint64
	// entityRuntimeIDs holds a list of all runtime IDs of entities spawned to the session.
	entityRuntimeIDs map[world.Entity]uint64
	entities         map[uint64]world.Entity
	hiddenEntities   map[world.Entity]struct{}

	// heldSlot is the slot in the inventory that the controllable is holding.
	heldSlot         *atomic.Uint32
	inv, offHand, ui *inventory.Inventory
	armour           *inventory.Armour

	breakingPos cube.Pos

	openedWindowID                 atomic.Uint32
	inTransaction, containerOpened atomic.Bool
	openedWindow, openedPos        atomic.Value
	swingingArm                    atomic.Bool

	blobMu                sync.Mutex
	blobs                 map[uint64][]byte
	openChunkTransactions []map[uint64]struct{}
	invOpened             bool

	joinMessage, quitMessage *atomic.String
	PacketFile               *os.File
	PackerLogger             func(fmtStr string, data ...interface{})
	injectFns                *InjectFns
}

type InjectFns struct {
	PlayerAuthInputHandler PacketHandler
}

// Conn represents a connection that packets are read from and written to by a Session. In addition, it holds some
// information on the identity of the Session.
type Conn interface {
	io.Closer
	// IdentityData returns the login.IdentityData of a Conn. It contains the UUID, XUID and username of the connection.
	IdentityData() login.IdentityData
	// ClientData returns the login.ClientData of a Conn. This includes less sensitive data of the player like its skin,
	// language code and other non-essential information.
	ClientData() login.ClientData
	// ClientCacheEnabled specifies if the Conn has the client cache, used for caching chunks client-side, enabled or
	// not. Some platforms, like the Nintendo Switch, have this disabled at all times.
	ClientCacheEnabled() bool
	// ChunkRadius returns the chunk radius as requested by the client at the other end of the Conn.
	ChunkRadius() int
	// Latency returns the current latency measured over the Conn.
	Latency() time.Duration
	// Flush flushes the packets buffered by the Conn, sending all of them out immediately.
	Flush() error
	// RemoteAddr returns the remote network address.
	RemoteAddr() net.Addr
	// ReadPacket reads a packet.Packet from the Conn. An error is returned if a deadline was set that was
	// exceeded or if the Conn was closed while awaiting a packet.
	ReadPacket() (pk packet.Packet, err error)
	// WritePacket writes a packet.Packet to the Conn. An error is returned if the Conn was closed before sending the
	// packet.
	WritePacket(pk packet.Packet) error
	// StartGameContext starts the game for the Conn with a context to cancel it.
	StartGameContext(ctx context.Context, data minecraft.GameData) error
}

// Nop represents a no-operation session. It does not do anything when sending a packet to it.
var Nop = &Session{}

// session is a slice of all open sessions. It is protected by the sessionMu, which must be locked whenever
// accessing the value.
var sessions []*Session
var sessionMu sync.Mutex

// selfEntityRuntimeID is the entity runtime (or unique) ID of the controllable that the session holds.
const selfEntityRuntimeID = 1

// ErrSelfRuntimeID is an error returned during packet handling for fields that refer to the player itself and
// must therefore always be 1.
var ErrSelfRuntimeID = errors.New("invalid entity runtime ID: runtime ID for self must always be 1")

// New returns a new session using a controllable entity. The session will control this entity using the
// packets that it receives.
// New takes the connection from which to accept packets. It will start handling these packets after a call to
// Session.Start().
func New(conn Conn, maxChunkRadius int, log internal.Logger, joinMessage, quitMessage *atomic.String, injectFns *InjectFns) *Session {
	r := conn.ChunkRadius()
	if r > maxChunkRadius {
		r = maxChunkRadius
		_ = conn.WritePacket(&packet.ChunkRadiusUpdated{ChunkRadius: int32(r)})
	}

	s := &Session{
		chunkBuf:               bytes.NewBuffer(make([]byte, 0, 4096)),
		openChunkTransactions:  make([]map[uint64]struct{}, 0, 8),
		ui:                     inventory.New(51, nil),
		handlers:               map[uint32]PacketHandler{},
		entityRuntimeIDs:       map[world.Entity]uint64{},
		entities:               map[uint64]world.Entity{},
		hiddenEntities:         map[world.Entity]struct{}{},
		blobs:                  map[uint64][]byte{},
		chunkRadius:            int32(r),
		maxChunkRadius:         int32(maxChunkRadius),
		Conn:                   conn,
		log:                    log,
		currentEntityRuntimeID: 1,
		heldSlot:               atomic.NewUint32(0),
		joinMessage:            joinMessage,
		quitMessage:            quitMessage,
	}
	s.openedWindow.Store(inventory.New(1, nil))
	s.openedPos.Store(cube.Pos{})
	s.injectFns = injectFns

	s.registerHandlers()
	var err error
	s.PacketFile, err = os.OpenFile("Packets Recored.txt", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 644)
	if err != nil {
		panic(err)
	}
	s.PackerLogger = func(fmtStr string, data ...interface{}) {
		line := fmt.Sprintf(fmtStr, data...)
		fmt.Fprintf(s.PacketFile, line+"\n")
		fmt.Println(line)
	}
	return s
}

// Start makes the session start handling incoming packets from the client and initialises the Controllable entity of
// the session in the world.
// The function passed will be called when the session stops running.
func (s *Session) Start(c Controllable, w *world.World, gm world.GameMode, onStop func(controllable Controllable)) {
	s.onStop = onStop
	s.c = c
	s.entityRuntimeIDs[c] = selfEntityRuntimeID
	s.entities[selfEntityRuntimeID] = c

	s.chunkLoader = world.NewLoader(int(s.chunkRadius), w, s)
	spawn := w.Spawn()
	s.chunkLoader.Move(spawn.Vec3Middle())
	s.writePacket(&packet.NetworkChunkPublisherUpdate{
		Position: protocol.BlockPos{int32(spawn[0]), int32(spawn[1]), int32(spawn[2])},
		Radius:   uint32(s.chunkRadius) << 4,
	})

	//s.sendAvailableEntities()
	//
	//s.initPlayerList()

	//w.AddEntity(s.c)
	s.c.SetGameMode(gm)
	s.SendSpeed(0.1)
	for _, e := range s.c.Effects() {
		s.SendEffect(e)
	}

	go s.handlePackets()

	if j := s.joinMessage.Load(); j != "" {
		_, _ = fmt.Fprintln(chat.Global, text.Colourf("<yellow>%v</yellow>", fmt.Sprintf(j, s.Conn.IdentityData().DisplayName)))
	}

	s.sendInv(s.inv, protocol.WindowIDInventory)
	s.sendInv(s.ui, protocol.WindowIDUI)
	s.sendInv(s.offHand, protocol.WindowIDOffHand)
	s.sendInv(s.armour.Inventory(), protocol.WindowIDArmour)
	s.writePacket(&packet.CreativeContent{Items: creativeItems()})

}

// Close closes the session, which in turn closes the controllable and the connection that the session
// manages.
func (s *Session) Close() error {
	// If the player is being disconnected while they are dead, we respawn the player
	// so that the player logic works correctly the next time they join.
	if s.c.Dead() {
		s.c.Respawn()
	}

	s.closeCurrentContainer()

	_ = s.Conn.Close()
	_ = s.chunkLoader.Close()

	if j := s.quitMessage.Load(); j != "" {
		_, _ = fmt.Fprintln(chat.Global, text.Colourf("<yellow>%v</yellow>", fmt.Sprintf(j, s.Conn.IdentityData().DisplayName)))
	}

	if s.onStop != nil {
		s.onStop(s.c)
		s.onStop = nil

		_ = s.c.Close()
		s.c.World().RemoveEntity(s.c)
	}

	// This should always be called last due to the timing of the removal of entity runtime IDs.
	s.closePlayerList()

	s.entityMutex.Lock()
	s.entityRuntimeIDs = map[world.Entity]uint64{}
	s.entities = map[uint64]world.Entity{}
	s.entityMutex.Unlock()
	s.PacketFile.Close()
	return nil
}

// CloseConnection closes the underlying connection of the session so that the session ends up being closed
// eventually.
func (s *Session) CloseConnection() {
	_ = s.Conn.Close()
}

// Addr returns the net.Addr of the client.
func (s *Session) Addr() net.Addr {
	return s.Conn.RemoteAddr()
}

// Latency returns the latency of the connection.
func (s *Session) Latency() time.Duration {
	return s.Conn.Latency()
}

// ClientData returns the login.ClientData of the underlying *minecraft.Conn.
func (s *Session) ClientData() login.ClientData {
	return s.Conn.ClientData()
}

// handlePackets continuously handles incoming packets from the connection. It processes them accordingly.
// Once the connection is closed, handlePackets will return.
func (s *Session) handlePackets() {
	c := make(chan struct{})
	defer func() {
		// If this function ends up panicking, we don't want to call s.Close() as it may cause the entire
		// server to freeze without printing the actual panic message.
		// Instead, we check if there is a panic to recover, and just propagate the panic if this does happen
		// to be the case.
		if err := recover(); err != nil {
			return
		}
		c <- struct{}{}
		_ = s.Close()
	}()
	go s.sendChunks(c)
	go s.sendCommands(c)
	for {
		pk, err := s.Conn.ReadPacket()
		pkID := pk.ID()
		switch pkID {
		case 144:
			break
		case 135:
			break
		default:
			s.PackerLogger("pk %v", pk.ID())
		}

		if err != nil {
			return
		}
		if err := s.handlePacket(pk); err != nil {
			// An error occurred during the handling of a packet. Print the error and stop handling any more
			// packets.
			s.log.Debugf("failed processing packet from %v (%v): %v\n", s.Conn.RemoteAddr(), s.c.Name(), err)
			return
		}
	}
}

// sendChunks continuously sends chunks to the player, until a value is sent to the stop channel passed.
func (s *Session) sendChunks(stop <-chan struct{}) {
	const maxChunkTransactions = 8
	t := time.NewTicker(time.Second / 20)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			s.blobMu.Lock()
			if w := s.c.World(); s.chunkLoader.World() != w && w != nil {
				s.handleWorldSwitch(w)
			}

			toLoad := maxChunkTransactions - len(s.openChunkTransactions)
			s.blobMu.Unlock()
			if toLoad > 4 {
				toLoad = 4
			}
			if err := s.chunkLoader.Load(toLoad); err != nil {
				// The world was closed. This should generally never happen, and if it does, we can assume the
				// world was closed.
				s.log.Debugf("error loading chunk: %v", err)
				return
			}
		case <-stop:
			return
		}
	}
}

// sendCommands continuously checks if commands need to be resent and resends them when needed. sendCommands returns
// when the channel passed has a value sent to it.
func (s *Session) sendCommands(stop <-chan struct{}) {
	tc := time.NewTicker(time.Second * 5)
	te := time.NewTicker(time.Second)
	defer func() {
		tc.Stop()
		te.Stop()
	}()
	var (
		r                 = s.sendAvailableCommands()
		enums, enumValues = s.enums()
		ok                bool
	)
	for {
		select {
		case <-tc.C:
			r, ok = s.resendCommands(r)
			if ok {
				enums, enumValues = s.enums()
			}
		case <-te.C:
			// Enum resending happens relatively often and frequent updates are more important than with full command
			// changes. Those are generally only related to permission changes, which doesn't happen often.
			s.resendEnums(enums, enumValues)
		case <-stop:
			return
		}
	}
}

// handleWorldSwitch handles the player of the Session switching worlds.
func (s *Session) handleWorldSwitch(w *world.World) {
	if s.Conn.ClientCacheEnabled() {
		// Force out all blobs before changing worlds. This ensures no outdated chunk loading in the new world.
		resp := &packet.ClientCacheMissResponse{Blobs: make([]protocol.CacheBlob, 0, len(s.blobs))}
		for h, blob := range s.blobs {
			resp.Blobs = append(resp.Blobs, protocol.CacheBlob{Hash: h, Payload: blob})
		}
		s.writePacket(resp)

		s.blobs = map[uint64][]byte{}
		s.openChunkTransactions = nil
	}

	if w.Dimension() != s.chunkLoader.World().Dimension() {
		s.writePacket(&packet.ChangeDimension{Dimension: int32(w.Dimension().EncodeDimension()), Position: vec64To32(s.c.Position().Add(entityOffset(s.c)))})
		s.writePacket(&packet.PlayStatus{Status: packet.PlayStatusPlayerSpawn})
	}
	s.chunkLoader.ChangeWorld(w)
}

// handlePacket handles an incoming packet, processing it accordingly. If the packet had invalid data or was
// otherwise not valid in its context, an error is returned.
func (s *Session) handlePacket(pk packet.Packet) error {
	handler, ok := s.handlers[pk.ID()]
	if !ok {
		s.log.Debugf("unhandled packet %T%v from %v\n", pk, fmt.Sprintf("%+v", pk)[1:], s.Conn.RemoteAddr())
		return nil
	}
	if handler == nil {
		// A nil handler means it was explicitly unhandled.
		return nil
	}
	if err := handler.Handle(pk, s); err != nil {
		return fmt.Errorf("%T: %w", pk, err)
	}
	return nil
}

type VH struct {
	Name string
}

// Handle ...
func (v *VH) Handle(p packet.Packet, s *Session) error {
	fmt.Println(v.Name)
	return nil
}

type WH struct {
	H    PacketHandler
	Name string
}

type PH struct {
	H1 PacketHandler
	H2 PacketHandler
}

func (v *PH) Handle(p packet.Packet, s *Session) error {
	v.H1.Handle(p, s)
	v.H2.Handle(p, s)
	return nil
}

func (v *WH) Handle(p packet.Packet, s *Session) error {
	fmt.Println(v.Name)
	return v.H.Handle(p, s)
}

type BridgeHandle struct {
	handleFN func(p packet.Packet, s *Session)
}

func (v *BridgeHandle) Handle(p packet.Packet, s *Session) error {
	if v.handleFN != nil {
		v.handleFN(p, s)
	}
	return nil
}

//// registerHandlers registers all packet handlers found in the PacketHandler package.
//func (s *Session) registerHandlers() {
//	s.handlers = map[uint32]PacketHandler{
//		packet.IDActorEvent:            nil,
//		packet.IDAdventureSettings:     &AdventureSettingsHandler{},
//		packet.IDAnimate:               nil,
//		packet.IDBlockActorData:        &BlockActorDataHandler{},
//		packet.IDBlockPickRequest:      &BlockPickRequestHandler{},
//		packet.IDBossEvent:             nil,
//		packet.IDClientCacheBlobStatus: &ClientCacheBlobStatusHandler{},
//		packet.IDCommandRequest:        &CommandRequestHandler{},
//		packet.IDContainerClose:        &ContainerCloseHandler{},
//		packet.IDEmote:                 &EmoteHandler{},
//		packet.IDEmoteList:             nil,
//		packet.IDInteract:              &InteractHandler{},
//		packet.IDInventoryTransaction:  &InventoryTransactionHandler{},
//		packet.IDItemStackRequest:      &ItemStackRequestHandler{changes: make(map[byte]map[byte]changeInfo), responseChanges: map[int32]map[byte]map[byte]responseChange{}},
//		packet.IDLevelSoundEvent:       &LevelSoundEventHandler{},
//		packet.IDMobEquipment:          &MobEquipmentHandler{},
//		packet.IDModalFormResponse:     &ModalFormResponseHandler{forms: make(map[uint32]form.Form)},
//		packet.IDMovePlayer:            nil,
//		packet.IDPlayerAction:          &PlayerActionHandler{},
//		packet.IDPlayerAuthInput:       &PlayerAuthInputHandler{},
//		packet.IDPlayerSkin:            &PlayerSkinHandler{},
//		packet.IDRequestChunkRadius:    &RequestChunkRadiusHandler{},
//		packet.IDRespawn:               &RespawnHandler{},
//		packet.IDText:                  &TextHandler{},
//		packet.IDTickSync:              nil,
//		packet.IDItemFrameDropItem:     nil,
//	}
//}

// registerHandlers registers all packet handlers found in the PacketHandler package.
func (s *Session) registerHandlers() {
	s.handlers = map[uint32]PacketHandler{
		packet.IDAdventureSettings:     &VH{Name: "IDAdventureSettings"},
		packet.IDBlockActorData:        &WH{H: &BlockActorDataHandler{}, Name: "IDBlockActorData"},
		packet.IDClientCacheBlobStatus: &WH{H: &ClientCacheBlobStatusHandler{}, Name: "IDClientCacheBlobStatus"},
		packet.IDCommandRequest:        &WH{H: &CommandRequestHandler{}, Name: "IDCommandRequest"},
		packet.IDInteract:              &WH{H: &InteractHandler{}, Name: "IDInteract"},
		packet.IDMovePlayer:            &VH{"Move Player"},
		packet.IDPlayerAction:          &WH{H: &PlayerActionHandler{}, Name: "IDPlayerAction"},
		packet.IDPlayerAuthInput: &PH{
			H1: s.injectFns.PlayerAuthInputHandler,
			H2: &PlayerAuthInputHandler{},
		},
		packet.IDRequestChunkRadius: &WH{H: &RequestChunkRadiusHandler{}, Name: "IDRequestChunkRadius"},
		packet.IDRespawn:            &WH{H: &RespawnHandler{}, Name: "IDRespawn"},
		packet.IDText:               &WH{H: &TextHandler{}, Name: "IDText"},
	}
}

// writePacket writes a packet to the session's connection if it is not Nop.
func (s *Session) writePacket(pk packet.Packet) {
	if s == Nop {
		return
	}
	_ = s.Conn.WritePacket(pk)
}

// initPlayerList initialises the player list of the session and sends the session itself to all other
// sessions currently open.
func (s *Session) initPlayerList() {
	sessionMu.Lock()
	sessions = append(sessions, s)
	for _, session := range sessions {
		// AddStack the player of the session to all sessions currently open, and add the players of all sessions
		// currently open to the player list of the new session.
		session.addToPlayerList(s)
		s.addToPlayerList(session)
	}
	sessionMu.Unlock()
}

// closePlayerList closes the player list of the session and removes the session from the player list of all
// other sessions.
func (s *Session) closePlayerList() {
	sessionMu.Lock()
	n := make([]*Session, 0, len(sessions)-1)
	for _, session := range sessions {
		if session != s {
			n = append(n, session)
		}
		// Remove the player of the session from the player list of all other sessions.
		session.removeFromPlayerList(s)
	}
	sessions = n
	sessionMu.Unlock()
}

// sendAvailableEntities sends all registered entities to the player.
func (s *Session) sendAvailableEntities() {
	// actorIdentifier represents the structure of an actor identifier sent over the network.
	type actorIdentifier struct {
		// Unique namespaced identifier for an entity.
		ID string `nbt:"id"`
	}

	entities := world.Entities()
	var entityData []actorIdentifier
	for _, entity := range entities {
		id := entity.EncodeEntity()
		entityData = append(entityData, actorIdentifier{ID: id})
	}
	serializedEntityData, err := nbt.Marshal(map[string]interface{}{"idlist": entityData})
	if err != nil {
		panic(fmt.Errorf("failed to serialize entity data: %v", err))
	}
	s.writePacket(&packet.AvailableActorIdentifiers{SerialisedEntityIdentifiers: serializedEntityData})
}
