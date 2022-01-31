package config

import (
	"flag"
	"main.go/minecraft/protocol/login"
	"main.go/shield"
)

var argAuthConfigFile = flag.String("c", "", "config file path")

type PluginConfig struct {
	Name    string      `yaml:"name"`
	As      string      `yaml:"as"`
	File    string      `yaml:"file"`
	Require []string    `yaml:"require"`
	Configs interface{} `yaml:"configs"`
}

type PluginSystemConfig struct {
	Version string         `yaml:"version"`
	Plugins []PluginConfig `yaml:"plugins"`
}
type InternationalMCConfig struct {
	Server            string             `json:"server_address"`
	ResponseUser      string             `json:"response_user"`
	LoginClientData   login.ClientData   `json:"login_client_data"`
	LoginIdentityData login.IdentityData `json:"login_identity_data"`
}
type FastBuilderMCConfig struct {
	MaskTermPassword bool `json:"mask_term_password"`
	// FB Login in config
	FBUserName       string `json:"user"`
	FBPassword       string `json:"password"`
	FBToken          string `json:"token"`
	UseFBVersion     string `json:"version"`
	FBCurrentVersion string `json:"current_version"`
	FBVersionCodeUrl string `json:"version_code_url"`
	// Server Config
	// we use this to judge whether the server code is taken from config or is from input, because we don't want to change the config
	origServerCode     string
	ServerCode         string `json:"server"`
	origServerPassword string // similar to origServerCode
	ServerPassword     string `json:"server_password"`
	NoPassword         bool   `json:"not_record_password"`
	NoToken            bool   `json:"not_record_token"`
}
type DragonFlyServerConfig struct {
}
type StartConfig struct {
	// the mc version between rental server and international be server is different
	// so it cannot be decided by user
	variant     int
	FBMCConfig  FastBuilderMCConfig   `json:"fb_mc_config"`
	IncMCConfig InternationalMCConfig `json:"international_mc_config"`
	// Shield Config
	ShieldConfig shield.ShieldConfig `json:"shield_config"`
	// Dragonfly Server Config
	ServerConfig DragonFlyServerConfig `json:"server_config"`
	// Plugin Config
	pluginsConfig    PluginSystemConfig
	PluginConfigPath string `json:"plugin_config_path"`
	// Aux
	writeBackPath string
}

func (s *StartConfig) GetVariant() int {
	return s.variant
}

func (s *StartConfig) GetPluginConfig() *PluginSystemConfig {
	return &s.pluginsConfig
}
