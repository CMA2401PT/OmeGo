package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
	"io"
	"main.go/auth/fb"
	"main.go/auth/inc"
	_const "main.go/const"
	"main.go/define"
	"main.go/minecraft/protocol/login"
	"main.go/plugins"
	"main.go/shield"
	"main.go/task"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
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

type StartConfig struct {
	// the mc version between rental server and international be server is different
	// so it cannot be decided by user
	variant     int
	FBMCConfig  FastBuilderMCConfig   `json:"fb_mc_config"`
	IncMCConfig InternationalMCConfig `json:"international_mc_config"`
	// Shield Config
	ShieldConfig shield.ShieldConfig `json:"shield_config"`
	// Plugin Config
	pluginsConfig    PluginSystemConfig
	PluginConfigPath string `json:"plugin_config_path"`
	// Aux
	writeBackPath string
}

func collectInfo() *StartConfig {
	flag.Parse()
	args := flag.Args()
	config := StartConfig{
		variant: _const.VARIANT,
		FBMCConfig: FastBuilderMCConfig{
			MaskTermPassword: true,
			UseFBVersion:     "use_current",
			ServerPassword:   "ask",
			FBVersionCodeUrl: "https://storage.fastbuilder.pro/hashes.json",
			NoPassword:       true,
			NoToken:          false,
		},
		IncMCConfig: InternationalMCConfig{
			Server:          "",
			ResponseUser:    "",
			LoginClientData: login.ClientData{},
			LoginIdentityData: login.IdentityData{
				DisplayName: "Bot",
			},
		},
		PluginConfigPath: "plugins_config.json",
		ShieldConfig: shield.ShieldConfig{
			Respawn:         true,
			MaxRetryTimes:   0,
			MaxDelaySeconds: 32,
		},
	}
	authConfigFile := *argAuthConfigFile
	config.writeBackPath = authConfigFile
	if authConfigFile == "" && len(args) > 0 {
		authConfigFile = args[0]
		args = args[1:]
		config.writeBackPath = authConfigFile
	} else {
		// 检查是否有默认配置文件 "config.json"
		_, err := os.Lstat("config.json")
		config.writeBackPath = "config.json"
		if os.IsNotExist(err) {
			fmt.Println("Main: No config provided, will create a config file automatically")
		} else {
			fmt.Println("Main: Config file not specific, we will use the default: 'config.json'")
			authConfigFile = "config.json"
		}
	}
	if authConfigFile != "" {
		// 读取配置文件
		fp, err := os.Open(authConfigFile)
		defer fp.Close()
		err = json.NewDecoder(fp).Decode(&config)
		if err != nil {
			panic(fmt.Sprintf("Main: Error at Unmarshal fbauth config file (%v) (%v)", authConfigFile, err))
		}
	}

	if config.variant == _const.Variant_Rental {
		fmt.Println("Rental Server Version: FB Config is required")
	} else {
		fmt.Println("International Bedrock Server version")
	}

	if config.variant == _const.Variant_Rental {
		// FB account
		if config.FBMCConfig.FBToken == "" && (config.FBMCConfig.FBUserName == "" || config.FBMCConfig.FBPassword == "") {
			// 询问用户名和密码
			reader := bufio.NewReader(os.Stdin)
			for config.FBMCConfig.FBUserName == "" {
				fmt.Printf("FB User Name: ")
				config.FBMCConfig.FBUserName, _ = reader.ReadString('\n')
				config.FBMCConfig.FBUserName = strings.TrimRight(config.FBMCConfig.FBUserName, "\r\n")
			}
			for config.FBMCConfig.FBPassword == "" {
				fmt.Printf("FB User Password: ")
				if config.FBMCConfig.MaskTermPassword {
					bytePassword, _ := term.ReadPassword(int(syscall.Stdin))
					config.FBMCConfig.FBPassword = strings.TrimSpace(string(bytePassword))
					fmt.Println("")
				} else {
					config.FBMCConfig.FBPassword, _ = reader.ReadString('\n')
					config.FBMCConfig.FBPassword = strings.TrimRight(config.FBMCConfig.FBPassword, "\r\n")
				}
			}
		}
		config.FBMCConfig.origServerCode = config.FBMCConfig.ServerCode
		config.FBMCConfig.origServerPassword = config.FBMCConfig.ServerPassword

		var serverCode, serverPassword string
		if len(args) > 1 {
			// get from command line args
			serverComplex := args[0]
			serverArgs := strings.Split(serverComplex, ":")
			serverCode = serverArgs[0]
			if len(serverArgs) > 1 {
				serverPassword = serverArgs[1]
			}
		} else {
			// get from config file
			serverCode = config.FBMCConfig.ServerCode
			serverPassword = config.FBMCConfig.ServerPassword
		}
		if serverCode == "" {
			// 询问服务器和密码
			reader := bufio.NewReader(os.Stdin)
			for serverCode == "" {
				fmt.Printf("Server Code: ")
				serverCode, _ = reader.ReadString('\n')
				serverCode = strings.TrimRight(serverCode, "\r\n")
				config.FBMCConfig.ServerCode = serverCode
			}

		}
		if serverPassword == "ask" {
			fmt.Printf("Server Password: ")
			if config.FBMCConfig.MaskTermPassword {
				bytePassword, _ := term.ReadPassword(int(syscall.Stdin))
				config.FBMCConfig.ServerPassword = strings.TrimSpace(string(bytePassword))
				fmt.Println("")
			} else {
				reader := bufio.NewReader(os.Stdin)
				serverPassword, _ = reader.ReadString('\n')
				serverPassword = strings.TrimRight(serverPassword, "\r\n")
				config.FBMCConfig.ServerPassword = serverPassword
			}
		}
	} else {
		// International
		if config.IncMCConfig.Server == "" {
			reader := bufio.NewReader(os.Stdin)
			serverAddr := ""
			for serverAddr == "" {
				fmt.Printf("Server Address (should be something like 127.0.0.1:19123): ")
				serverAddr, _ = reader.ReadString('\n')
				serverAddr = strings.TrimRight(serverAddr, "\r\n")
				config.IncMCConfig.Server = serverAddr
			}
		}
		if config.IncMCConfig.ResponseUser == "" {
			reader := bufio.NewReader(os.Stdin)
			ResponseUser := ""
			for ResponseUser == "" {
				fmt.Printf("Who Should Bot Response?: ")
				ResponseUser, _ = reader.ReadString('\n')
				ResponseUser = strings.TrimRight(ResponseUser, "\r\n")
				config.IncMCConfig.ResponseUser = ResponseUser
			}
		}
	}
	// load plugins config file
	fp, err := os.Open(config.PluginConfigPath)
	defer fp.Close()
	err = yaml.NewDecoder(fp).Decode(&config.pluginsConfig)
	if err != nil {
		panic(fmt.Sprintf("Main: Error at Unmarshal plugin config file (%v) (%v)", config.PluginConfigPath, err))
	}
	return &config
}

func obtainPageContent(pageUrl string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(pageUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var buffer [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}
	return result.Bytes(), nil
}

func getVersionInfo(versionCodeUrl string) (string, error) {
	fmt.Printf("FB Version: Request version code from page : (%v)\n", versionCodeUrl)
	pageContent, err := obtainPageContent(versionCodeUrl, 3*time.Second)
	if err != nil {
		return "", err
	}
	availableVersion := make([]string, 0)
	err = json.Unmarshal(pageContent, &availableVersion)
	if err != nil {
		return "", err
	}
	version := availableVersion[0]
	fmt.Printf("FB Version: Version update success : (%v)\n", version)
	return version, nil
}

func writeBackConfig(config *StartConfig) {
	copiedConfig := *config
	if config.FBMCConfig.NoPassword {
		copiedConfig.FBMCConfig.FBPassword = ""
	}
	if config.FBMCConfig.NoToken {
		copiedConfig.FBMCConfig.FBToken = ""
	}
	copiedConfig.FBMCConfig.ServerCode = config.FBMCConfig.origServerCode
	copiedConfig.FBMCConfig.ServerPassword = config.FBMCConfig.origServerPassword

	fp, err := os.Create(copiedConfig.writeBackPath)
	defer fp.Close()
	if err != nil {
		panic(fmt.Sprintf("Main: Fail to create updated config (%v)", err))
	}
	encoder := json.NewEncoder(fp)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "\t")
	encoder.Encode(copiedConfig)
	if err != nil {
		panic(fmt.Sprintf("Main: fail to marshal updated config (%v)", err))
	}
}

func updateFBConfig(config FastBuilderMCConfig) FastBuilderMCConfig {
	if config.UseFBVersion == "auto_update" || (config.UseFBVersion == "use_current" && config.FBCurrentVersion == "") {
		version, err := getVersionInfo(config.FBVersionCodeUrl)
		if err != nil {
			panic(fmt.Errorf("Main: Fail to fectch FB version (%v)", err))
		}
		config.FBCurrentVersion = version
	}
	if config.FBToken == "" {
		fbClient, err := fb.CreateClient()
		if err != nil {
			panic(fmt.Errorf("Main: When update FB token, fail to create FB client (%v)", err))
		}
		fbToken, err := fbClient.GetToken(config.FBUserName, config.FBPassword)
		if err != nil {
			panic(fmt.Errorf("Main: When update FB token, update fbToken (%v)", err))
		}
		config.FBToken = fbToken
		fbClient.Close()
	}
	return config
}

func main() {
	color.Blue("Collecting Infomation...")
	config := collectInfo()
	if config.variant == _const.Variant_Rental {
		config.FBMCConfig = updateFBConfig(config.FBMCConfig)
	}
	writeBackConfig(config)
	color.Green("Information Collected!")

	//storage := initStorage(&config.StorageConfig)
	//defer storage.Close()

	color.Blue("Starting Shield...")
	var authenticator define.Authenticator
	if config.variant == _const.Variant_Rental {
		authenticator = &fb.Authenticator{
			Identify: &fb.Identify{
				FBToken:        config.FBMCConfig.FBToken,
				FBVersion:      config.FBMCConfig.FBCurrentVersion,
				ServerCode:     config.FBMCConfig.ServerCode,
				ServerPassword: config.FBMCConfig.ServerPassword,
			},
		}
	} else {
		authenticator = &inc.Authenticator{Address: config.IncMCConfig.Server}
	}

	mcShield := shield.NewShield(&config.ShieldConfig)
	taskIO := task.NewTaskIO(mcShield.IO)

	mcShield.LoginTokenGenerator = authenticator.GenerateToken
	mcShield.PacketInterceptor = authenticator.Intercept

	mcShield.Variant = config.variant
	if mcShield.Variant == _const.Variant_Inc {
		mcShield.LoginClientData = config.IncMCConfig.LoginClientData
		mcShield.LoginIdentityData = config.IncMCConfig.LoginIdentityData
	}

	closeFn := loadPlugins(taskIO, &config.pluginsConfig)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// make sure data are saved
	go func() {
		s := <-c
		fmt.Println("Got signal:", s)
		closeFn()
		fmt.Println("Close Functions done")
		os.Exit(0)
	}()

	mcShield.Routine()
}

func loadPlugins(taskIO *task.TaskIO, config *PluginSystemConfig) func() {
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
