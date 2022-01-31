package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
	"io"
	_const "main.go/const"
	"main.go/minecraft/protocol/login"
	"main.go/shield"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
)

func CollectInfo() *StartConfig {
	flag.Parse()
	args := flag.Args()
	config := StartConfig{
		variant: _const.VARIANT,
		FBMCConfig: FastBuilderMCConfig{
			MaskTermPassword: true,
			UseFBVersion:     "use_current",
			ServerPassword:   "ask",
			FBVersionCodeUrl: "https://storage.fastbuilder.pro/epsilon/hashes.json",
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
		ServerConfig:     DragonFlyServerConfig{},
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

func GetVersionInfo(versionCodeUrl string) (string, error) {
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

func WriteBackConfig(config *StartConfig) {
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
