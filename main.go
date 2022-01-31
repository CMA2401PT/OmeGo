package main

import (
	"fmt"
	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
	auth_define "main.go/auth/define"
	"main.go/auth/fb"
	"main.go/auth/inc"
	config "main.go/config"
	_const "main.go/const"
	"main.go/plugins"
	define "main.go/plugins/define"
	"main.go/shield"
	"main.go/task"
	WordInit "main.go/world/init"
	"os"
	"os/signal"
)

func updateFBConfig(cfg config.FastBuilderMCConfig) config.FastBuilderMCConfig {
	if cfg.UseFBVersion == "auto_update" || (cfg.UseFBVersion == "use_current" && cfg.FBCurrentVersion == "") {
		version, err := config.GetVersionInfo(cfg.FBVersionCodeUrl)
		if err != nil {
			panic(fmt.Errorf("Main: Fail to fectch FB version (%v)", err))
		}
		cfg.FBCurrentVersion = version
	}
	if cfg.FBToken == "" {
		fbClient, err := fb.CreateClient()
		if err != nil {
			panic(fmt.Errorf("Main: When update FB token, fail to create FB client (%v)", err))
		}
		fbToken, err := fbClient.GetToken(cfg.FBUserName, cfg.FBPassword)
		if err != nil {
			panic(fmt.Errorf("Main: When update FB token, update fbToken (%v)", err))
		}
		cfg.FBToken = fbToken
		fbClient.Close()
	}
	return cfg
}

func main() {
	color.Blue("Collecting Infomation...")
	cfg := config.CollectInfo()
	if cfg.GetVariant() == _const.Variant_Rental {
		cfg.FBMCConfig = updateFBConfig(cfg.FBMCConfig)
	}
	config.WriteBackConfig(cfg)
	color.Green("Information Collected!")

	//storage := initStorage(&config.StorageConfig)
	//defer storage.Close()

	color.Blue("Starting Shield...")
	var authenticator auth_define.Authenticator
	if cfg.GetVariant() == _const.Variant_Rental {
		authenticator = &fb.Authenticator{
			Identify: &fb.Identify{
				FBToken:        cfg.FBMCConfig.FBToken,
				FBVersion:      cfg.FBMCConfig.FBCurrentVersion,
				ServerCode:     cfg.FBMCConfig.ServerCode,
				ServerPassword: cfg.FBMCConfig.ServerPassword,
			},
		}
	} else {
		authenticator = &inc.Authenticator{Address: cfg.IncMCConfig.Server}
	}

	mcShield := shield.NewShield(&cfg.ShieldConfig)
	taskIO := task.NewTaskIO(mcShield.IO)
	taskIO.StartConfig = cfg

	mcShield.LoginTokenGenerator = authenticator.GenerateToken
	mcShield.PacketInterceptor = authenticator.Intercept

	mcShield.Variant = cfg.GetVariant()
	if mcShield.Variant == _const.Variant_Inc {
		mcShield.LoginClientData = cfg.IncMCConfig.LoginClientData
		mcShield.LoginIdentityData = cfg.IncMCConfig.LoginIdentityData
	}
	serverCloseFn := initDragonFlyServer(taskIO, &cfg.ServerConfig)
	pluginCloseFn := loadPlugins(taskIO, cfg.GetPluginConfig())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// make sure data are saved
	go func() {
		s := <-c
		fmt.Println("Got signal:", s)
		serverCloseFn()
		pluginCloseFn()
		fmt.Println("Close Functions done")
		os.Exit(0)
	}()

	mcShield.Routine()
}

func initDragonFlyServer(task *task.TaskIO, config *config.DragonFlyServerConfig) func() {
	WordInit.InitRuntimeIds()
	return func() {}
}

func loadPlugins(taskIO *task.TaskIO, config *config.PluginSystemConfig) func() {
	if config.Version != "0.0.0" {
		panic("Main-loadPlugins: Version Not Support!")
	}
	closeFns := make([]func(), 0)
	collaborationContext := make(map[string]define.Plugin)
	for i, plugin := range config.Plugins {
		if plugin.As == "" {
			plugin.As = plugin.Name
		}
		color.Blue("loading Plugin: %v. %v As %v from %v", i, plugin.Name, plugin.As, plugin.File)
		if plugin.Require != nil {
			for _, r := range plugin.Require {
				_, hasK := collaborationContext[r]
				if !hasK {
					panic(fmt.Sprintf(`plugin: %v require plugin: "%v", but "%v" has not injected!`, plugin.Name, r, r))
				}
			}
		}
		pluginConfigBytes, _ := yaml.Marshal(plugin.Configs)
		var pi define.Plugin
		if plugin.File == "internal" {
			p, ok := plugins.Pool()[plugin.Name]
			if !ok {
				panic(color.New(color.FgRed).Sprintf("No Such file Plugin: (%v)", plugin.Name))
			} else {
				pi = p().New(pluginConfigBytes)
			}
		} else {
			if plugin.File == "" {
				panic(fmt.Sprintf("loading Plugin: file path of (%v) not specific!", plugin.Name))
			}
		}
		collaborationContext[plugin.As] = pi
		go pi.Inject(taskIO, collaborationContext).Routine()
		closeFns = append(closeFns, func() { pi.Close() })
	}
	return func() {
		for _, fn := range closeFns {
			fn()
		}
	}
}
