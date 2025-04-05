package main

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

var cfgName = ".omnienv.yaml"

var errCfgNotFound = errors.New("Config not found")

type System struct {
	Name  string
	Image string
}

func NewSystem(val string) System {
	return System{Name: val}
}

func (sys *System) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// if string, unmarshal to Name and done
	var err error
	var name string

	if err = unmarshal(&name); err == nil {
		*sys = NewSystem(name)
		return nil
	}

	type image struct {
		Image string
	}

	// if map with single key, unmarshal key to Name and set Image
	var dict map[string]image
	if err = unmarshal(&dict); err == nil {
		for name, img := range dict {
			*sys = System{Name: name, Image: img.Image}
		}
		return nil
	}

	return err
}

type Config struct {
	// System indicates what distribution and version to base this upon.
	// Specifying distribution not yet implemented.
	// When distribution is omitted, Ubuntu is used.
	System System
	// Label is set to the basename of RootDir by default, or may be
	// overwritten with the supplied value.  Used as part of the name of
	// the instance, along with System.
	Label string
	// RootDir is the writable base directory that the container has access
	// to.  This field is optional, and if unsupplied uses the parent
	// directory of the omnienv.yaml config.
	RootDir string
	// Backend indicates upon what we are running the instance.
	// Only "lxd" is implemented.
	Backend string
	// Virtualization chooses between "container" (default) and "vm".
	Virtualization string

	// unsupported keys that are unmarshalled for warning purposes
	Project string
	Series  string
}

func (cfg Config) IsVM() bool {
	return cfg.Virtualization == "vm"
}

func (cfg Config) LXDLaunchConfig() string {
	tmap := map[string]string{
		"WORKDIR": cfg.RootDir,
	}
	template := `
devices:
  home:
    type: disk
    shift: true
    path: /home
    source: /home
    readonly: true
  workdir:
    type: disk
    readonly: false
    shift: true
    path: ${WORKDIR}
    source: ${WORKDIR}
`
	return os.Expand(template, func(key string) string {
		return tmap[key]
	})
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func findConfig(dir string) (string, error) {
	for {
		cur := dir + "/" + cfgName
		if exists(cur) {
			return cur, nil
		}
		if dir == "/" {
			break
		}
		dir = filepath.Dir(dir)
	}

	return "", errCfgNotFound
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.RootDir == "" {
		cfg.RootDir = filepath.Dir(path)
	}

	if cfg.Label == "" {
		cfg.Label = filepath.Base(cfg.RootDir)
	}

	if cfg.System.Name == "" {
		cfg.System = NewSystem(os.Getenv("DEFAULT_SERIES"))
	}

	if cfg.Virtualization == "" {
		cfg.Virtualization = "container"
	}

	if cfg.Project != "" {
		slog.Warn("unsupported key", "project", cfg.Project)
	}
	if cfg.Series != "" {
		slog.Warn("unsupported key", "series", cfg.Series)
	}
	slog.Debug("loadConfig", "config", cfg)
	return cfg, nil
}

func GetConfig() (Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}

	cfgPath, err := findConfig(dir)
	if err != nil {
		return Config{}, err
	}

	return loadConfig(cfgPath)
}
