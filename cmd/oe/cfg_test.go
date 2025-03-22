package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var cfgTests = []struct {
	summary string
	config  Config

	name string
	vm   bool
}{{
	summary: "basic name",
	config: Config{
		Label:  "l",
		Series: "s",
	},
	name: "l-s",
	vm:   false,
}, {
	summary: "vm",
	config: Config{
		Label:          "foo",
		Series:         "bar",
		Virtualization: "vm",
	},
	name: "foo-bar",
	vm:   true,
}}

func TestCfg(t *testing.T) {
	for _, test := range cfgTests {
		assert.Equal(t, test.name, test.config.Name(), test.summary)
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

var loadCfgTests = []struct {
	summary   string
	data      string
	seriesEnv string

	config Config
}{{
	summary: "series",
	data:    "series: plucky",
	config: Config{
		Series:         "plucky",
		Virtualization: "container",
	},
}, {
	summary: "series / label",
	data: `
series: warty
label: ubiquity
`,
	config: Config{
		Series:         "warty",
		Label:          "ubiquity",
		Virtualization: "container",
	},
}, {
	summary: "series environ / lxd / default virt",
	data:    "backend: lxd",
	config: Config{
		Series:         "zesty",
		Backend:        "lxd",
		Virtualization: "container",
	},
}, {
	summary: "series environ / lxd / vm",
	data: `
backend: lxd
virtualization: vm
`,
	config: Config{
		Series:         "zesty",
		Backend:        "lxd",
		Virtualization: "vm",
	},
}}

func TestLoadCfg(t *testing.T) {
	tempdir := t.TempDir()
	dirname := tempdir + "/foo"
	assert.Nil(t, os.Mkdir(dirname, 0750))
	filename := dirname + "/" + cfgName

	os.Setenv("DEFAULT_SERIES", "zesty")

	for _, test := range loadCfgTests {
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

	data := []byte("series: warty")
	filename := tempdir + "/" + cfgName
	assert.Nil(t, os.WriteFile(filename, data, 0644))
	actual, err := GetConfig()
	assert.Nil(t, err)
	assert.Equal(t, "warty", actual.Series)
}

func TestNotGetConfig(t *testing.T) {
	curdir, err := os.Getwd()
	assert.Nil(t, err)
	assert.Nil(t, os.Chdir("/"))
	defer func() { _ = os.Chdir(curdir) }()

	_, err = GetConfig()
	assert.Equal(t, errCfgNotFound, err)
}
