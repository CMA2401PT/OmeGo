package plugins

import (
	"encoding/json"
	"fmt"
	"log"
	"main.go/task"
	"os"
	"path"
)

type StorageConfig struct {
	Root   string `json:"root"`
	Logs   string `json:"logs"`
	ROJson string `json:"ro_json"`
	DB     string `json:"db"`
}

type Storage struct {
	logRoot    string
	roJsonRoot string
	dbRoot     string
	closeFn    []func()
}

func (s *Storage) New(config []byte) Plugin {
	storageConfig := &StorageConfig{}
	err := json.Unmarshal(config, storageConfig)
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

func (s *Storage) Inject(taskIO *task.TaskIO, collaborationContext map[string]Plugin) Plugin {
	return s
}

func (s *Storage) Close() {
	for _, fn := range s.closeFn {
		fn()
	}
}

func (s *Storage) RegStringSender(source string) func(isJson bool, data string) {
	fileName := path.Join(s.logRoot, source) + ".log"
	logFile, err := os.Open(fileName)
	if err != nil && os.IsNotExist(err) {
		logFile, err = os.Create(fileName)
		if err != nil {
			panic(fmt.Sprintf("Storage-Create: cannot create %v (%v)", fileName, err))
		}
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
