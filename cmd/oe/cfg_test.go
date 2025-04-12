package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var cfgTests = []struct {
	summary string
	config  Config

	vm bool
}{{
	summary: "basic name",
	config:  Config{Label: "l", Systems: []System{NewSystem("s")}},
	vm:      false,
}, {
	summary: "vm",
	config:  Config{Label: "foo", Systems: []System{NewSystem("bar")}, Virtualization: "vm"},
	vm:      true,
}}

func TestCfg(t *testing.T) {
	for _, test := range cfgTests {
		assert.Equal(t, test.vm, test.config.IsVM(), test.summary)
	}
}

func TestFindCfg(t *testing.T) {
	tempdir := t.TempDir()
	assert.Nil(t, os.WriteFile(tempdir+"/"+cfgName, []byte{}, 0644))
	_, err := findConfig(tempdir)
	assert.Nil(t, err)
}

func TestFindParentCfg(t *testing.T) {
	tempdir := t.TempDir()
	dirname := tempdir + "/foo"
	assert.Nil(t, os.Mkdir(dirname, 0750))
	assert.Nil(t, os.WriteFile(tempdir+"/"+cfgName, []byte{}, 0644))
	_, err := findConfig(dirname)
	assert.Nil(t, err)
}

func patchEnv(key string, mock string) func() {
	original, ok := os.LookupEnv(key)
	_ = os.Setenv(key, mock)
	return func() {
		if ok {
			_ = os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	}
}

var loadCfgTests = []struct {
	summary   string
	data      string
	seriesEnv string

	config Config
	warn   string
	image  string
}{{
	summary: "systems",
	data:    "systems: [plucky]",
	config: Config{
		Systems:        []System{NewSystem("plucky")},
		Virtualization: "container",
	},
}, {
	summary: "system / label",
	data: `
systems: [warty]
label: ubiquity
`,
	config: Config{
		Systems:        []System{NewSystem("warty")},
		Label:          "ubiquity",
		Virtualization: "container",
	},
}, {
	summary: "system environ / lxd / default virt",
	data:    "backend: lxd",
	config: Config{
		Systems:        []System{NewSystem("zesty")},
		Backend:        "lxd",
		Virtualization: "container",
	},
}, {
	summary: "system environ / lxd / vm",
	data: `
backend: lxd
virtualization: vm
`,
	config: Config{
		Systems:        []System{NewSystem("zesty")},
		Backend:        "lxd",
		Virtualization: "vm",
	},
	image: "ubuntu-daily:zesty",
}, {
	summary: "project",
	data:    "project: proj",
	config: Config{
		Project:        "proj",
		Systems:        []System{NewSystem("zesty")},
		Virtualization: "container",
	},
	warn: `msg="unsupported key"`,
}, {
	summary: "series",
	data:    "series: warty",
	config: Config{
		Series:         "warty",
		Systems:        []System{NewSystem("zesty")},
		Virtualization: "container",
	},
	warn: `msg="unsupported key"`,
}, {
	summary: "manual remote for image",
	data: `
systems:
    - jammy:
        image: ubuntu:j
`,

	config: Config{
		Systems:        []System{System{"jammy", "ubuntu:j"}},
		Virtualization: "container",
	},
	image: "ubuntu:j",
}}

func TestLoadCfg(t *testing.T) {
	tempdir := t.TempDir()
	dirname := tempdir + "/foo"
	assert.Nil(t, os.Mkdir(dirname, 0750))
	filename := dirname + "/" + cfgName

	restore, buf := patchLogger()
	defer restore()
	setupLogging(Opts{})

	restoreEnv := patchEnv("DEFAULT_SERIES", "zesty")
	defer restoreEnv()

	for _, test := range loadCfgTests {
		buf.Reset()
		err := os.WriteFile(filename, []byte(test.data), 0644)
		assert.Nil(t, err, test.summary)
		actual, err := loadConfig(filename)
		assert.Nil(t, err, test.summary)
		if test.config.RootDir == "" {
			test.config.RootDir = dirname
		}
		if test.config.Label == "" {
			test.config.Label = "foo"
		}
		assert.Equal(t, test.config, actual, test.summary)
		if len(test.warn) > 0 {
			assert.Contains(t, buf.String(), test.warn, test.summary)
		}

		if test.image != "" {
			assert.Equal(
				t, test.config.Systems[0].LaunchImage(),
				test.image, test.summary,
			)
		}
	}
}

func TestNotLoadCfgUnreadable(t *testing.T) {
	tempdir := t.TempDir()
	filename := tempdir + "/" + cfgName
	assert.Nil(t, os.WriteFile(filename, []byte{}, 000))
	_, err := loadConfig(filename)
	assert.NotNil(t, err)
}

func TestNotLoadCfgUnmarshalable(t *testing.T) {
	tempdir := t.TempDir()
	data := []byte(`{`)
	filename := tempdir + "/" + cfgName
	assert.Nil(t, os.WriteFile(filename, data, 0644))
	_, err := loadConfig(filename)
	assert.NotNil(t, err)
}

func TestGetConfig(t *testing.T) {
	tempdir := t.TempDir()

	curdir, err := os.Getwd()
	assert.Nil(t, err)
	assert.Nil(t, os.Chdir(tempdir))
	defer func() { _ = os.Chdir(curdir) }()

	data := []byte("system: warty")
	filename := tempdir + "/" + cfgName
	assert.Nil(t, os.WriteFile(filename, data, 0644))
	actual, err := GetConfig()
	assert.Nil(t, err)
	assert.Equal(t, "warty", actual.System.Name)
}

func TestNotGetConfig(t *testing.T) {
	curdir, err := os.Getwd()
	assert.Nil(t, err)
	assert.Nil(t, os.Chdir("/"))
	defer func() { _ = os.Chdir(curdir) }()

	_, err = GetConfig()
	assert.Equal(t, errCfgNotFound, err)
}

func TestLXDLaunchConfigHomeAndWorkdir(t *testing.T) {
	restoreHome := patchEnv("HOME", "/tmp/a")
	defer restoreHome()

	restoreUser := patchEnv("USER", "jimbob")
	defer restoreUser()

	cfg := Config{RootDir: "/tmp/b"}
	expected := `
config:
  user.vendor-data: |
    #cloud-config
    users:
      - name: jimbob
        sudo: ALL=(ALL) NOPASSWD:ALL
        groups: users,admin
        shell: /bin/bash
devices:
  home:
    type: disk
    readonly: true
    shift: true
    path: /tmp/a
    source: /tmp/a
  workdir:
    type: disk
    readonly: false
    shift: true
    path: /tmp/b
    source: /tmp/b`
	assert.Equal(t, expected, cfg.LXDLaunchConfig())
}

func TestLXDLaunchConfigWorkdirOnly(t *testing.T) {
	restoreHome := patchEnv("HOME", "/tmp/b")
	defer restoreHome()

	restoreUser := patchEnv("USER", "jimbob")
	defer restoreUser()

	cfg := Config{RootDir: "/tmp/b"}
	expected := `
config:
  user.vendor-data: |
    #cloud-config
    users:
      - name: jimbob
        sudo: ALL=(ALL) NOPASSWD:ALL
        groups: users,admin
        shell: /bin/bash
devices:
  workdir:
    type: disk
    readonly: false
    shift: true
    path: /tmp/b
    source: /tmp/b`
	assert.Equal(t, expected, cfg.LXDLaunchConfig())
}
