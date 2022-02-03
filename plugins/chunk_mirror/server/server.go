package server

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"
	"main.go/minecraft"
	"main.go/minecraft/nbt"
	"main.go/minecraft/protocol"
	"main.go/minecraft/protocol/login"
	"main.go/minecraft/protocol/packet"
	"main.go/minecraft/resource"
	"main.go/minecraft/text"
	"main.go/plugins/chunk_mirror/server/block"
	"main.go/plugins/chunk_mirror/server/cmd"
	"main.go/plugins/chunk_mirror/server/internal"
	_ "main.go/plugins/chunk_mirror/server/item" // Imported for compiler directives.
	"main.go/plugins/chunk_mirror/server/player"
	"main.go/plugins/chunk_mirror/server/player/playerdb"
	"main.go/plugins/chunk_mirror/server/player/skin"
	"main.go/plugins/chunk_mirror/server/session"
	"main.go/plugins/chunk_mirror/server/world"
	"main.go/plugins/chunk_mirror/server/world/biome"
	"main.go/plugins/chunk_mirror/server/world/generator"
	"main.go/plugins/chunk_mirror/server/world/mcdb"
)

// Server implements a Dragonfly server. It runs the main server loop and handles the connections of players
// trying to join the server.
type Server struct {
	started atomic.Bool
	name    atomic.String

	joinMessage, quitMessage atomic.String
	playerProvider           player.Provider

	c                  Config
	log                internal.Logger
	world, nether, end *world.World
	players            chan *player.Player
	resources          []*resource.Pack

	startTime time.Time

	playerMutex sync.RWMutex
	// p holds a map of all players currently connected to the server. When they leave, they are removed from
	// the map.
	p map[uuid.UUID]*player.Player
	// pwg is a sync.WaitGroup used to wait for all players to be disconnected before server shutdown, so that their
	// data is saved properly.
	pwg sync.WaitGroup

	wg sync.WaitGroup

	listenMu  sync.Mutex
	listeners []Listener

	aMu sync.Mutex
	a   Allower
}

func init() {
	// Seeding the random for things like lightning that need to use RNG.
	rand.Seed(time.Now().UnixNano())
}

// New returns a new server using the Config passed. If nil is passed, a default configuration is returned.
// (A call to server.DefaultConfig().)
// The Logger passed will be used to log errors and information to. If nil is passed, a default Logger is
// used by calling logrus.New().
// Note that no two servers should be active at the same time. Doing so anyway will result in unexpected
// behaviour.
func New(c *Config, log internal.Logger) *Server {
	if log == nil {
		log = logrus.New()
	}
	log.Infof("Loading server...")
	if c == nil {
		conf := DefaultConfig()
		c = &conf
	}
	s := &Server{
		c:              *c,
		log:            log,
		players:        make(chan *player.Player),
		p:              make(map[uuid.UUID]*player.Player),
		name:           *atomic.NewString(c.Server.Name),
		playerProvider: player.NopProvider{},
		a:              allower{},
	}
	set := new(world.Settings)
	s.world = s.createWorld(world.Overworld, biome.Plains{}, []world.Block{block.Grass{}, block.Dirt{}, block.Dirt{}, block.Bedrock{}}, set)
	s.nether = s.createWorld(world.Nether, biome.NetherWastes{}, []world.Block{block.Netherrack{}, block.Netherrack{}, block.Netherrack{}, block.Bedrock{}}, set)
	s.end = s.createWorld(world.End, biome.End{}, []world.Block{block.EndStone{}, block.EndStone{}, block.EndStone{}, block.Bedrock{}}, set)

	s.world.SetPortalDestinations(s.nether, s.end)
	s.nether.SetPortalDestinations(s.world, s.end)
	s.end.SetPortalDestinations(s.nether, s.world)

	s.registerTargetFunc()

	s.JoinMessage(c.Server.JoinMessage)
	s.QuitMessage(c.Server.QuitMessage)

	s.loadResources(c.Resources.Folder, log)
	s.checkNetIsolation()

	if c.Players.SaveData {
		p, err := playerdb.NewProvider(c.Players.Folder)
		if err != nil {
			log.Fatalf("error loading player provider: %v", err)
		}
		s.PlayerProvider(p)
	}
	return s
}

// Accept accepts an incoming player into the server. It blocks until a player connects to the server.
// Accept returns an error if the Server is closed using a call to Close.
func (server *Server) Accept() (*player.Player, error) {
	p, ok := <-server.players
	if !ok {
		return nil, errors.New("server closed")
	}
	server.playerMutex.Lock()
	server.p[p.UUID()] = p
	server.playerMutex.Unlock()

	return p, nil
}

// World returns the overworld of the server. Players will be spawned in this world and this world will be read
// from and written to when the world is edited.
func (server *Server) World() *world.World {
	return server.world
}

// Nether returns the nether world of the server. Players are transported to it when entering a nether portal in the
// world returned by the World method.
func (server *Server) Nether() *world.World {
	return server.nether
}

// End returns the end world of the server. Players are transported to it when entering an end portal in the world
// returned by the World method.
func (server *Server) End() *world.World {
	return server.end
}

// Start runs the server but does not block, unlike Run, but instead accepts connections on a different
// goroutine. Connections will be accepted until the listener is closed using a call to Close.
// Once started, players may be accepted using Server.Accept().
func (server *Server) Start() error {
	if !server.started.CAS(false, true) {
		panic("server already running")
	}

	server.log.Infof("Starting Dragonfly for Minecraft v%v...", protocol.CurrentVersion)
	if err := server.startListening(); err != nil {
		return err
	}
	go server.wait()
	return nil
}

// Uptime returns the duration that the server has been running for. Measurement starts the moment a call to
// Server.Start or Server.Run is made.
func (server *Server) Uptime() time.Duration {
	if !server.running() {
		return 0
	}
	return time.Since(server.startTime)
}

// PlayerCount returns the current player count of the server. It is equivalent to calling
// len(server.Players()).
func (server *Server) PlayerCount() int {
	server.playerMutex.RLock()
	defer server.playerMutex.RUnlock()

	return len(server.p)
}

// MaxPlayerCount returns the maximum amount of players that are allowed to play on the server at the same
// time. Players trying to join when the server is full will be refused to enter.
// If the config has a maximum player count set to 0, MaxPlayerCount will return Server.PlayerCount + 1.
func (server *Server) MaxPlayerCount() int {
	if server.c.Players.MaxCount == 0 {
		return server.PlayerCount() + 1
	}
	return server.c.Players.MaxCount
}

// Players returns a list of all players currently connected to the server. Note that the slice returned is
// not updated when new players join or leave, so it is only valid for as long as no new players join or
// players leave.
func (server *Server) Players() []*player.Player {
	server.playerMutex.RLock()
	defer server.playerMutex.RUnlock()

	players := make([]*player.Player, 0, len(server.p))
	for _, p := range server.p {
		players = append(players, p)
	}
	return players
}

// Player looks for a player on the server with the UUID passed. If found, the player is returned and the bool
// returns holds a true value. If not, the bool returned is false and the player is nil.
func (server *Server) Player(uuid uuid.UUID) (*player.Player, bool) {
	server.playerMutex.RLock()
	defer server.playerMutex.RUnlock()
	p, ok := server.p[uuid]
	return p, ok
}

// PlayerByName looks for a player on the server with the name passed. If found, the player is returned and the bool
// returns holds a true value. If not, the bool is false and the player is nil
func (server *Server) PlayerByName(name string) (*player.Player, bool) {
	for _, p := range server.Players() {
		if p.Name() == name {
			return p, true
		}
	}
	return nil, false
}

// PlayerProvider changes the data provider of a player to the provider passed. The provider will dictate
// the behaviour of player saving and loading. If nil is passed, the NopProvider will be used
// which does not read or write any data.
func (server *Server) PlayerProvider(provider player.Provider) {
	if provider == nil {
		provider = player.NopProvider{}
	}
	server.playerProvider = provider
}

// AddResourcePack loads a resource pack to the server. The pack will eventually be sent to clients who join the
// server when started.
func (server *Server) AddResourcePack(pack *resource.Pack) {
	server.resources = append(server.resources, pack)
}

// SetName sets the name of the Server, also known as the MOTD. This name is displayed in the server list.
// The formatting of the name passed follows the rules of fmt.Sprint.
func (server *Server) SetName(a ...interface{}) {
	server.name.Store(format(a))
}

// JoinMessage changes the join message for all players on the server. Leave this empty to disable it.
// %v is the placeholder for the username of the player
func (server *Server) JoinMessage(a ...interface{}) {
	server.joinMessage.Store(format(a))
}

// QuitMessage changes the leave message for all players on the server. Leave this empty to disable it.
// %v is the placeholder for the username of the player
func (server *Server) QuitMessage(a ...interface{}) {
	server.quitMessage.Store(format(a))
}

// Close closes the server, making any call to Run/Accept cancel immediately.
func (server *Server) Close() error {
	if !server.running() {
		panic("server not yet running")
	}

	server.log.Infof("Server shutting down...")
	defer server.log.Infof("Server stopped.")

	server.log.Debugf("Disconnecting players...")
	server.playerMutex.RLock()
	for _, p := range server.p {
		p.Disconnect(text.Colourf("<yellow>%v</yellow>", server.c.Server.ShutdownMessage))
	}
	server.playerMutex.RUnlock()
	server.pwg.Wait()

	server.log.Debugf("Closing player provider...")
	err := server.playerProvider.Close()
	if err != nil {
		server.log.Errorf("Error while closing player provider: %v", err)
	}

	server.log.Debugf("Closing worlds...")
	if err = server.world.Close(); err != nil {
		server.log.Errorf("Error closing overworld: %v", err)
	}
	if err = server.nether.Close(); err != nil {
		server.log.Errorf("Error closing nether %v", err)
	}
	if err = server.end.Close(); err != nil {
		server.log.Errorf("Error closing end: %v", err)
	}

	server.log.Debugf("Closing listeners...")
	server.listenMu.Lock()
	defer server.listenMu.Unlock()
	for _, l := range server.listeners {
		if err := l.Close(); err != nil {
			server.log.Errorf("Error closing listener: %v", err)
		}
	}
	return nil
}

// Allow makes the Server filter which connections to the Server are accepted. Connections on which the Allower returns
// false are rejected immediately. If nil is passed, all connections are accepted.
func (server *Server) Allow(a Allower) {
	if a == nil {
		a = allower{}
	}
	server.aMu.Lock()
	defer server.aMu.Unlock()
	server.a = a
}

// Listen makes the Server listen for new connections from the Listener passed. This may be used to listen for players
// on different interfaces. Note that the maximum player count of additional Listeners added is not enforced
// automatically. The limit must be enforced by the Listener.
func (server *Server) Listen(l Listener) {
	server.listenMu.Lock()
	server.listeners = append(server.listeners, l)
	server.listenMu.Unlock()

	server.wg.Add(1)

	wg := new(sync.WaitGroup)
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		for {
			c, err := l.Accept()
			if err != nil {
				// Cancel the context so that any call to StartGameContext is cancelled rapidly.
				cancel()
				// First wait until all connections that are being handled are done inserting the player into the channel.
				// Afterwards, when we're sure no more values will be inserted in the players channel, we can return so the
				// player channel can be closed.
				wg.Wait()
				server.wg.Done()
				return
			}
			server.aMu.Lock()
			a := server.a
			server.aMu.Unlock()

			if msg, ok := a.Allow(c.RemoteAddr(), c.IdentityData(), c.ClientData()); !ok {
				_ = c.WritePacket(&packet.Disconnect{HideDisconnectionScreen: msg == "", Message: msg})
				_ = c.Close()
				continue
			}
			wg.Add(1)
			go server.finaliseConn(ctx, c, l, wg)
		}
	}()
}

// CloseOnProgramEnd closes the server right before the program ends, so that all data of the server are
// saved properly.
func (server *Server) CloseOnProgramEnd() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		if err := server.Close(); err != nil {
			server.log.Errorf("error shutting down server: %v", err)
		}
	}()
}

// running checks if the server is currently running.
func (server *Server) running() bool {
	return server.started.Load()
}

// startListening starts making the EncodeBlock listener listen, accepting new connections from players.
func (server *Server) startListening() error {
	server.startTime = time.Now()

	cfg := minecraft.ListenConfig{
		MaximumPlayers:         server.c.Players.MaxCount,
		StatusProvider:         statusProvider{s: server},
		AuthenticationDisabled: !server.c.Server.AuthEnabled,
		ResourcePacks:          server.resources,
		Biomes:                 server.biomes(),
	}

	l, err := cfg.Listen("raknet", server.c.Network.Address)
	if err != nil {
		return fmt.Errorf("listening on address failed: %w", err)
	}
	server.Listen(listener{Listener: l})

	server.log.Infof("Server running on %v.\n", l.Addr())
	return nil
}

// wait awaits the closing of all Listeners added to the Server through a call to Listen and closed the players channel
// once that happens.
func (server *Server) wait() {
	server.wg.Wait()
	close(server.players)
}

// finaliseConn finalises the session.Conn passed and subtracts from the sync.WaitGroup once done.
func (server *Server) finaliseConn(ctx context.Context, conn session.Conn, l Listener, wg *sync.WaitGroup) {
	defer wg.Done()

	// UUID is validated by gophertunnel.
	id, _ := uuid.Parse(conn.IdentityData().Identity)
	data := server.defaultGameData()

	var playerData *player.Data
	if d, err := server.playerProvider.Load(id); err == nil {
		data.PlayerPosition = vec64To32(d.Position).Add(mgl32.Vec3{0, 1.62})
		data.Yaw, data.Pitch = float32(d.Yaw), float32(d.Pitch)
		data.Dimension = int32(server.dimension(d.Dimension).Dimension().EncodeDimension())

		playerData = &d
	}

	if err := conn.StartGameContext(ctx, data); err != nil {
		_ = l.Disconnect(conn, "Connection timeout.")
		server.log.Debugf("connection %v failed spawning: %v\n", conn.RemoteAddr(), err)
		return
	}
	if p, ok := server.Player(id); ok {
		p.Disconnect("Logged in from another location.")
	}
	server.players <- server.createPlayer(id, conn, playerData)
}

// defaultGameData returns a minecraft.GameData as sent for a new player. It may later be modified if the player was
// saved in the player provide rof the server.
func (server *Server) defaultGameData() minecraft.GameData {
	return minecraft.GameData{
		Yaw:            90,
		WorldName:      server.world.Name(),
		PlayerPosition: vec64To32(server.world.Spawn().Vec3Centre().Add(mgl64.Vec3{0, 1.62})),
		PlayerGameMode: 1,
		// We set these IDs to 1, because that's how the session will treat them.
		EntityUniqueID:               1,
		EntityRuntimeID:              1,
		Time:                         int64(server.world.Time()),
		GameRules:                    []protocol.GameRule{{Name: "naturalregeneration", Value: false}},
		Difficulty:                   2,
		Items:                        server.itemEntries(),
		PlayerMovementSettings:       protocol.PlayerMovementSettings{MovementType: protocol.PlayerMovementModeServer, ServerAuthoritativeBlockBreaking: true},
		ServerAuthoritativeInventory: true,
	}
}

// dimension returns a world by a dimension ID passed.
func (server *Server) dimension(id int) *world.World {
	switch id {
	default:
		return server.world
	case 1:
		return server.nether
	case 2:
		return server.end
	}
}

// checkNetIsolation checks if a loopback exempt is in place to allow the hosting device to join the server. This is
// only relevant on Windows. It will never log anything for anything but Windows.
func (server *Server) checkNetIsolation() {
	if runtime.GOOS != "windows" {
		// Only an issue on Windows.
		return
	}
	data, _ := exec.Command("CheckNetIsolation", "LoopbackExempt", "-s", `-n="microsoft.minecraftuwp_8wekyb3d8bbwe"`).CombinedOutput()
	if bytes.Contains(data, []byte("microsoft.minecraftuwp_8wekyb3d8bbwe")) {
		return
	}
	const loopbackExemptCmd = `CheckNetIsolation LoopbackExempt -a -n="Microsoft.MinecraftUWP_8wekyb3d8bbwe"`
	server.log.Infof("You are currently unable to join the server on this machine. Run %v in an admin PowerShell session to be able to.\n", loopbackExemptCmd)
}

// handleSessionClose handles the closing of a session. It removes the player of the session from the server.
func (server *Server) handleSessionClose(c session.Controllable) {
	server.playerMutex.Lock()
	p := server.p[c.UUID()]
	delete(server.p, c.UUID())
	server.playerMutex.Unlock()
	err := server.playerProvider.Save(p.UUID(), p.Data())
	if err != nil {
		server.log.Errorf("Error while saving data: %v", err)
	}
	server.pwg.Done()
}

// createPlayer creates a new player instance using the UUID and connection passed.
func (server *Server) createPlayer(id uuid.UUID, conn session.Conn, data *player.Data) *player.Player {
	w, gm, pos := server.world, server.world.DefaultGameMode(), server.world.Spawn().Vec3Middle()
	if data != nil {
		w, gm, pos = server.dimension(data.Dimension), data.GameMode, data.Position
	}
	s := session.New(conn, server.c.Players.MaximumChunkRadius, server.log, &server.joinMessage, &server.quitMessage)
	p := player.NewWithSession(conn.IdentityData().DisplayName, conn.IdentityData().XUID, id, server.createSkin(conn.ClientData()), s, pos, data)

	s.Start(p, w, gm, server.handleSessionClose)
	server.pwg.Add(1)
	return p
}

// createWorld loads a world of the server with a specific dimension, ending the program if the world could not be loaded.
// The layers passed are used to create a generator.Flat that is used as generator for the world.
func (server *Server) createWorld(d world.Dimension, biome world.Biome, layers []world.Block, s *world.Settings) *world.World {
	log := server.log
	if v, ok := log.(interface {
		WithField(key string, field interface{}) *logrus.Entry
	}); ok {
		// Add a dimension field to be able to distinguish between the different dimensions in the log. Dimensions
		// implement fmt.Stringer so we can just fmt.Sprint them for a readable name.
		log = v.WithField("dimension", strings.ToLower(fmt.Sprint(d)))
	}
	log.Debugf("Loading world...")

	w := world.New(log, d, s)

	p, err := mcdb.New(server.c.World.Folder, d)
	if err != nil {
		log.Fatalf("error loading world: %v", err)
	}
	w.Provider(p)
	w.Generator(generator.NewFlat(biome, layers))

	log.Debugf(`Loaded world "%v".`, w.Name())
	return w
}

// createSkin creates a new skin using the skin data found in the client data in the login, and returns it.
func (server *Server) createSkin(data login.ClientData) skin.Skin {
	// Gophertunnel guarantees the following values are valid data and are of the correct size.
	skinData, _ := base64.StdEncoding.DecodeString(data.SkinData)
	capeData, _ := base64.StdEncoding.DecodeString(data.CapeData)
	modelData, _ := base64.StdEncoding.DecodeString(data.SkinGeometry)
	skinResourcePatch, _ := base64.StdEncoding.DecodeString(data.SkinResourcePatch)
	modelConfig, _ := skin.DecodeModelConfig(skinResourcePatch)

	playerSkin := skin.New(data.SkinImageWidth, data.SkinImageHeight)
	playerSkin.Persona = data.PersonaSkin
	playerSkin.Pix = skinData
	playerSkin.Model = modelData
	playerSkin.ModelConfig = modelConfig
	playerSkin.PlayFabID = data.PlayFabID

	playerSkin.Cape = skin.NewCape(data.CapeImageWidth, data.CapeImageHeight)
	playerSkin.Cape.Pix = capeData

	for _, animation := range data.AnimatedImageData {
		var t skin.AnimationType
		switch animation.Type {
		case protocol.SkinAnimationHead:
			t = skin.AnimationHead
		case protocol.SkinAnimationBody32x32:
			t = skin.AnimationBody32x32
		case protocol.SkinAnimationBody128x128:
			t = skin.AnimationBody128x128
		}

		anim := skin.NewAnimation(animation.ImageWidth, animation.ImageHeight, animation.AnimationExpression, t)
		anim.FrameCount = int(animation.Frames)
		anim.Pix, _ = base64.StdEncoding.DecodeString(animation.Image)

		playerSkin.Animations = append(playerSkin.Animations, anim)
	}

	return playerSkin
}

// registerTargetFunc registers a cmd.TargetFunc to be able to get all players connected and all entities in
// the server's world.
func (server *Server) registerTargetFunc() {
	cmd.AddTargetFunc(func(src cmd.Source) ([]cmd.Target, []cmd.Target) {
		entities, players := src.World().Entities(), server.Players()
		eTargets, pTargets := make([]cmd.Target, len(entities)), make([]cmd.Target, len(players))

		for i, e := range entities {
			eTargets[i] = e
		}
		for i, p := range players {
			pTargets[i] = p
		}
		return eTargets, pTargets
	})
}

// vec64To32 converts a mgl64.Vec3 to a mgl32.Vec3.
func vec64To32(vec3 mgl64.Vec3) mgl32.Vec3 {
	return mgl32.Vec3{float32(vec3[0]), float32(vec3[1]), float32(vec3[2])}
}

// itemEntries loads a list of all custom item entries of the server, ready to be sent in the StartGame
// packet.
func (server *Server) itemEntries() (entries []protocol.ItemEntry) {
	for name, rid := range itemRuntimeIDs {
		entries = append(entries, protocol.ItemEntry{
			Name:      name,
			RuntimeID: int16(rid),
		})
	}
	return
}

// ashyBiome represents a biome that has any form of ash.
type ashyBiome interface {
	// Ash returns the ash and white ash of the biome.
	Ash() (ash float64, whiteAsh float64)
}

// sporingBiome represents a biome that has blue or red spores.
type sporingBiome interface {
	// Spores returns the blue and red spores of the biome.
	Spores() (blueSpores float64, redSpores float64)
}

// biomes builds a mapping of all biome definitions of the server, ready to be set in the biomes field of the
// server listener.
func (server *Server) biomes() map[string]interface{} {
	definitions := make(map[string]interface{})
	for _, b := range world.Biomes() {
		definition := map[string]interface{}{
			"temperature": float32(b.Temperature()),
			"downfall":    float32(b.Rainfall()),
		}
		if a, ok := b.(ashyBiome); ok {
			ash, whiteAsh := a.Ash()
			definition["ash"], definition["white_ash"] = float32(ash), float32(whiteAsh)
		}
		if s, ok := b.(sporingBiome); ok {
			blueSpores, redSpores := s.Spores()
			definition["blue_spores"], definition["red_spores"] = float32(blueSpores), float32(redSpores)
		}
		definitions[b.String()] = definition
	}
	return definitions
}

// loadResources loads resource packs from path of specifed directory.
func (server *Server) loadResources(p string, log internal.Logger) {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		_ = os.Mkdir(p, 0777)
	}
	resources, err := os.ReadDir(p)
	if err != nil {
		panic(err)
	}
	for _, entry := range resources {
		pack, err := resource.Compile(filepath.Join(p, entry.Name()))
		if err != nil {
			log.Infof("Failed to load resource: %v", entry.Name())
			continue
		}

		server.AddResourcePack(pack)
	}
}

// format is a utility function to format a list of values to have spaces between them, but no newline at the
// end, which is typically used for sending messages, popups and tips.
func format(a []interface{}) string {
	return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintln(a...), "\n"), "\n")
}

var (
	//go:embed world/item_runtime_ids.nbt
	itemRuntimeIDData []byte
	itemRuntimeIDs    = map[string]int32{}
)

// init reads all item entries from the resource JSON, and sets the according values in the runtime ID maps.
func init() {
	_ = nbt.Unmarshal(itemRuntimeIDData, &itemRuntimeIDs)
}
