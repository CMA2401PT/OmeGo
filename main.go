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
	"main.go/define"
	"main.go/fbauth"
	"main.go/plugins"
	"main.go/shield"
	"main.go/task"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
)

var argAuthConfigFile = flag.String("c", "", "config file path")

type WriteBackConfig struct {
	NoPassword    bool `json:"no_password"`
	NoToken       bool `json:"no_token"`
	writeBackPath string
}

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

type StartConfig struct {
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
	// Shield Config
	ShieldConfig shield.ShieldConfig `json:"shield_config"`
	// Plugin Config
	pluginsConfig    PluginSystemConfig
	PluginConfigPath string `json:"plugin_config_path"`
	// Aux
	WriteBackConfig WriteBackConfig `json:"write_back"`
}

func collectInfo() *StartConfig {
	flag.Parse()
	args := flag.Args()
	config := StartConfig{
		MaskTermPassword: true,
		UseFBVersion:     "use_current",
		ServerPassword:   "ask",
		FBVersionCodeUrl: "https://storage.fastbuilder.pro/hashes.json",
		PluginConfigPath: "plugins_config.json",
		ShieldConfig: shield.ShieldConfig{
			Respawn:         true,
			MaxRetryTimes:   0,
			MaxDelaySeconds: 32,
		},
		WriteBackConfig: WriteBackConfig{
			NoPassword: true,
			NoToken:    false,
		},
	}
	authConfigFile := *argAuthConfigFile
	config.WriteBackConfig.writeBackPath = authConfigFile
	if authConfigFile == "" && len(args) > 0 {
		authConfigFile = args[0]
		args = args[1:]
		config.WriteBackConfig.writeBackPath = authConfigFile
	} else {
		// 检查是否有默认配置文件 "config.json"
		_, err := os.Lstat("config.json")
		config.WriteBackConfig.writeBackPath = "config.json"
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

	// account
	if config.FBToken == "" && (config.FBUserName == "" || config.FBPassword == "") {
		// 询问用户名和密码
		reader := bufio.NewReader(os.Stdin)
		for config.FBUserName == "" {
			fmt.Printf("FB User Name: ")
			config.FBUserName, _ = reader.ReadString('\n')
			config.FBUserName = strings.TrimRight(config.FBUserName, "\r\n")
		}
		for config.FBPassword == "" {
			fmt.Printf("FB User Password: ")
			if config.MaskTermPassword {
				bytePassword, _ := term.ReadPassword(int(syscall.Stdin))
				config.FBPassword = strings.TrimSpace(string(bytePassword))
				fmt.Println("")
			} else {
				config.FBPassword, _ = reader.ReadString('\n')
				config.FBPassword = strings.TrimRight(config.FBPassword, "\r\n")
			}
		}
	}
	config.origServerCode = config.ServerCode
	config.origServerPassword = config.ServerPassword

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
		serverCode = config.ServerCode
		serverPassword = config.ServerPassword
	}
	if serverCode == "" {
		// 询问服务器和密码
		reader := bufio.NewReader(os.Stdin)
		for serverCode == "" {
			fmt.Printf("Server Code: ")
			serverCode, _ = reader.ReadString('\n')
			serverCode = strings.TrimRight(serverCode, "\r\n")
			config.ServerCode = serverCode
		}

	}
	if serverPassword == "ask" {
		fmt.Printf("Server Password: ")
		if config.MaskTermPassword {
			bytePassword, _ := term.ReadPassword(int(syscall.Stdin))
			config.ServerPassword = strings.TrimSpace(string(bytePassword))
			fmt.Println("")
		} else {
			reader := bufio.NewReader(os.Stdin)
			serverPassword, _ = reader.ReadString('\n')
			serverPassword = strings.TrimRight(serverPassword, "\r\n")
			config.ServerPassword = serverPassword
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
	if config.WriteBackConfig.NoPassword {
		copiedConfig.FBPassword = ""
	}
	if config.WriteBackConfig.NoToken {
		copiedConfig.FBToken = ""
	}
	copiedConfig.ServerCode = config.origServerCode
	copiedConfig.ServerPassword = config.origServerPassword
	fp, err := os.Create(copiedConfig.WriteBackConfig.writeBackPath)
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

func main() {
	color.Blue("Collecting Infomation...")
	config := collectInfo()
	if config.UseFBVersion == "auto_update" || (config.UseFBVersion == "use_current" && config.FBCurrentVersion == "") {
		version, err := getVersionInfo(config.FBVersionCodeUrl)
		if err != nil {
			panic(fmt.Errorf("Main: Fail to fectch FB version (%v)", err))
		}
		config.FBCurrentVersion = version
	}
	if config.FBToken == "" {
		fbClient, err := fbauth.CreateClient()
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
	writeBackConfig(config)
	color.Green("Information Collected!")

	//storage := initStorage(&config.StorageConfig)
	//defer storage.Close()

	color.Blue("Starting Shield...")
	authenticator := &fbauth.Authenticator{
		Identify: &fbauth.Identify{
			FBToken:        config.FBToken,
			FBVersion:      config.FBCurrentVersion,
			ServerCode:     config.ServerCode,
			ServerPassword: config.ServerPassword,
		},
	}

	mcShield := shield.NewShield(&config.ShieldConfig)
	taskIO := task.NewTaskIO(mcShield.IO)
	mcShield.LoginTokenGenerator = authenticator.GenerateToken
	mcShield.PacketInterceptor = authenticator.Intercept

	closeFn := loadPlugins(taskIO, &config.pluginsConfig)
	defer closeFn()

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
