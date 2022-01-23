package plugins

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"main.go/define"
	"main.go/task"
	"os"
	"path"
)

type StorageConfig struct {
	Root   string `yaml:"root"`
	Logs   string `yaml:"logs"`
	ROJson string `yaml:"ro_json"`
	DB     string `yaml:"db"`
}

type Storage struct {
	logRoot    string
	roJsonRoot string
	dbRoot     string
	closeFn    []func()
}

func (s *Storage) New(config []byte) define.Plugin {
	storageConfig := &StorageConfig{}
	err := yaml.Unmarshal(config, storageConfig)
	if err != nil {
		panic(err)
	}
	if storageConfig.Root == "" {
		storageConfig.Root = "data"
	}
	if storageConfig.Logs == "" {
		storageConfig.Logs = path.Join(storageConfig.Root, "logs")
	}
	if storageConfig.ROJson == "" {
		storageConfig.ROJson = path.Join(storageConfig.Root, "jsons")
	}
	if storageConfig.DB == "" {
		storageConfig.DB = path.Join(storageConfig.Root, "db")
	}
	return s.initStorage(storageConfig)
}

func (s *Storage) Routine() {

}

func (s *Storage) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	return s
}

func (s *Storage) Close() {
	for _, fn := range s.closeFn {
		fn()
	}
}

func (s *Storage) RegStringSender(source string) func(isJson bool, data string) {
	fileName := path.Join(s.logRoot, source) + ".log"
	logFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 644)
	if err != nil && os.IsNotExist(err) {
		panic(fmt.Sprintf("Storage-Create: cannot create or append %v (%v)", fileName, err))
	}
	s.closeFn = append(s.closeFn, func() {
		logFile.Close()
	})
	log_ := log.New(logFile, "", log.Ldate|log.Ltime)
	return func(isJson bool, data string) {
		if isJson {
			var anyData interface{}
			err := json.Unmarshal([]byte(data), &anyData)
			if err == nil {
				log_.Printf("(%v) Json> %v", source, anyData)
			} else {
				log_.Printf("(%v) BrokenJson(%v)> %v", source, err, data)
			}
		} else {
			log_.Printf("(%v) > %v", source, data)
		}
	}
}

func (s *Storage) initStorage(config *StorageConfig) *Storage {
	err := os.MkdirAll(config.Logs, 644)
	if err != nil {
		panic(fmt.Sprintf("Main-InitStorage: cannot create %v (%v)", config.Logs, err))
	}
	err = os.MkdirAll(config.ROJson, 644)
	if err != nil {
		panic(fmt.Sprintf("Main-InitStorage: cannot create %v (%v)", config.ROJson, err))
	}
	err = os.MkdirAll(config.DB, 644)
	if err != nil {
		panic(fmt.Sprintf("Main-InitStorage: cannot create %v (%v)", config.DB, err))
	}
	ret := &Storage{logRoot: config.Logs, roJsonRoot: config.ROJson, dbRoot: config.DB,
		closeFn: make([]func(), 0)}
	return ret
}
