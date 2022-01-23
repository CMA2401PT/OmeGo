package plugins

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"main.go/define"
	"main.go/task"
	"os"
	"path"
)

type StorageConfig struct {
	Root string `yaml:"root"`
	Logs string `yaml:"logs"`
	Cfgs string `yaml:"ro_cfg"`
	DB   string `yaml:"db"`
}

type Storage struct {
	logRoot  string
	CfgsRoot string
	dbRoot   string
	closeFn  []func()
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
	if storageConfig.Cfgs == "" {
		storageConfig.Cfgs = path.Join(storageConfig.Root, "ro_cfg")
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

func (s *Storage) getCfgPath(file string) ([]byte, bool) {
	full_path := path.Join(s.CfgsRoot, file)
	fp, err := os.OpenFile(full_path, os.O_RDONLY, 0)
	defer fp.Close()
	if err != nil {
		panic(fmt.Sprintf("Storage: not a readable file %v (%v)", full_path, err))
	}
	fd, err := ioutil.ReadAll(fp)
	if err != nil {
		panic(fmt.Sprintf("Storage: cannot read data from %v (%v)", full_path, err))
	}
	return fd, true
}

func (s *Storage) OpenSqlite(file string) (*sql.DB, error) {
	fullPath := path.Join(s.dbRoot, file)
	db, err := sql.Open("sqlite3", fullPath+".db")
	if err != nil {
		panic(fmt.Sprintf("Storage: Cannot Open Sqlite DB: %v (%v)", fullPath, err))
	}
	s.closeFn = append(s.closeFn, func() {
		fmt.Println("Storage: %v close", fullPath)
		db.Close()
	})
	return db, nil
}

func (s *Storage) initStorage(config *StorageConfig) *Storage {
	err := os.MkdirAll(config.Logs, 644)
	if err != nil {
		panic(fmt.Sprintf("Main-InitStorage: cannot create %v (%v)", config.Logs, err))
	}
	err = os.MkdirAll(config.Cfgs, 644)
	if err != nil {
		panic(fmt.Sprintf("Main-InitStorage: cannot create %v (%v)", config.Cfgs, err))
	}
	err = os.MkdirAll(config.DB, 644)
	if err != nil {
		panic(fmt.Sprintf("Main-InitStorage: cannot create %v (%v)", config.DB, err))
	}
	ret := &Storage{logRoot: config.Logs, CfgsRoot: config.Cfgs, dbRoot: config.DB,
		closeFn: make([]func(), 0)}
	return ret
}
