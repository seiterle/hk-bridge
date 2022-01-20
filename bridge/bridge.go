package bridge

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"gopkg.in/yaml.v2"
)

const (
	rootPath   = "./.hr"
	hcPath     = "hc-store"
	configFile = "config.yaml"
)

type Bridge struct {
	bridge      *accessory.Bridge
	accessories []*accessory.Accessory

	Name     string `yaml:"name"`
	Password string `yaml:"password"`

	Pin string `yaml:"pin"` // HomeKit PIN
}

func loadConfig() (*Bridge, error) {
	// support yml file extension by changing it to yaml
	os.Rename("config.yml", configFile)

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	bridge := Bridge{}
	err = yaml.Unmarshal(data, &bridge)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %v", err)
	}

	info := accessory.Info{
		Name: bridge.Name,
	}
	bridge.bridge = accessory.NewBridge(info)

	return &bridge, nil
}

// update will update the content of the config file
func (b *Bridge) update() error {
	data, err := yaml.Marshal(b)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configFile, data, 666)
}

// NewBridge creates a new bridge according to local configuration.
func NewBridge(as ...*accessory.Accessory) (*Bridge, error) {

	// create and move to workdir
	path, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(path, 0755)
	if err != nil {
		return nil, err
	}
	err = os.Chdir(path)
	if err != nil {
		return nil, err
	}

	bridge, err := loadConfig()
	if err != nil {
		// TODO: create new config?
		fmt.Println("handle empty config")
	}

	if bridge.Name == "" {
		bridge.Name = "Bridge " + randString(6)
		fmt.Println("Name: " + bridge.Name)
	}
	if bridge.Password == "" {
		bridge.Password = randString(12)
		fmt.Println("Password: " + bridge.Password)
	}
	if bridge.Pin == "" { // TODO: check if valid homekit pin
		bridge.Pin = randPin()
		fmt.Println("HomeKit Pin: " + bridge.Pin)
	}

	bridge.update()

	return bridge, nil
}

// Start the bridge
func (b *Bridge) Start() {
	config := hc.Config{
		Pin:         b.Pin,
		StoragePath: hcPath,
	}
	t, err := hc.NewIPTransport(config, b.bridge.Accessory, b.accessories...)
	if err != nil {
		log.Fatal(err) // TODO: should dispaly to user via web console instead of crashing
	}
	hc.OnTermination(func() {
		<-t.Stop()
	})
	t.Start()
}
