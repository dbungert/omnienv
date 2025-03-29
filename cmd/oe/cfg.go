package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

var cfgName = ".omnienv.yaml"

var errCfgNotFound = errors.New("Config not found")

type Config struct {
	// System indicates what distribution and version to base this upon.
	System string
	// RootDir is the parent directory of the .omnienv.yaml file.
	RootDir string
	// Label is set to the basename of RootDir by default, or may be
	// overwritten with the supplied value.  Used as part of the name of
	// the instance, along with System.
	Label string
	// Backend indicates upon what we are running the instance.
	// Only "lxd" is implemented.
	Backend string
	// Virtualization chooses between "container" (default) and "vm".
	Virtualization string

	// unsupported keys that are unmarshalled for warning purposes
	Project string
	Series  string
}

func (cfg Config) Name() string {
	return fmt.Sprintf("%s-%s", cfg.Label, cfg.System)
}

func (cfg Config) IsVM() bool {
	return cfg.Virtualization == "vm"
}

func (cfg Config) LXDLaunchConfig() string {
	return `
devices:
  home:
    type: disk
    shift: true
    path: /home
    source: /home
`
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

	if cfg.System == "" {
		cfg.System = os.Getenv("DEFAULT_SERIES")
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
