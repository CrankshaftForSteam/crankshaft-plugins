package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
)

type InputPlugin struct {
	Id               string `json:"id"`
	Repo             string `json:"repo"`
	PluginConfigPath string `json:"pluginConfigPath"`

	Version string `json:"version"`
	Archive string `json:"archive"`
	Sha256  string `json:"sha256"`

	Source string `json:"source"`
	Name   string `json:"name"`
}

type InputFile struct {
	Plugins []InputPlugin `json:"plugins"`
}

const INPUT_FILE = "plugins.json"

type author struct {
	Name string `toml:"name" json:"name"`
	Link string `toml:"link" json:"link"`
}

type platform struct {
	Supported bool `toml:"supported" json:"supported"`
}

type platforms struct {
	Linux   platform `toml:"linux" json:"linux"`
	Windows platform `toml:"windows" json:"windows"`
	Darwin     platform `toml:"darwin" json:"darwin"`
}

type store struct {
	Description string    `toml:"description" json:"description"`
	Platforms   platforms `toml:"platforms" json:"platforms"`
}

type PluginConfig struct {
	Name                 string `toml:"name" json:"name"`
	Version              string `toml:"version" json:"version"`
	Link                 string `toml:"link" json:"link"`
	Source               string `toml:"source" json:"source"`
	MinCrankshaftVersion string `toml:"min-crankshaft-version" json:"minCrankshaftVersion"`

	Author author `toml:"author" json:"author"`

	Store store `toml:"store" json:"store"`
}

type OutputPlugin struct {
	Id                   string `json:"id"`
	Name                 string `json:"name"`
	Version              string `json:"version"`
	Archive              string `json:"archive"`
	Sha256               string `json:"sha256"`
	MinCrankshaftVersion string `json:"minCrankshaftVersion"`

	Link   string `json:"link"`
	Source string `json:"source"`

	Author author `json:"author"`

	Store store `json:"store"`
}

type OutputPlugins map[string]OutputPlugin

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	inputBytes, err := os.ReadFile(INPUT_FILE)
	if err != nil {
		return err
	}

	var input InputFile

	if err := json.Unmarshal(inputBytes, &input); err != nil {
		return err
	}

	inputPlugins := input.Plugins
	outputPlugins := make(OutputPlugins)

	downloadsDir, err := ioutil.TempDir("", "plugin-downloads")
	if err != nil {
		return err
	}
	defer os.RemoveAll(downloadsDir)

	for _, plugin := range inputPlugins {
		downloadPath := path.Join(downloadsDir, plugin.Id+".tar.gz")
		fmt.Printf("%s: Downloading to %s\n", plugin.Id, downloadPath)
		err := exec.Command("wget", plugin.Archive, "-O", downloadPath).Run()
		if err != nil {
			return err
		}

		fmt.Printf("%s: Validating checksum\n", plugin.Id)
		sha256Bytes, err := exec.Command("sha256sum", downloadPath).Output()
		if err != nil {
			return err
		}
		sha256 := strings.Split(strings.TrimSpace(string(sha256Bytes)), " ")[0]
		if strings.ToLower(sha256) != strings.ToLower(plugin.Sha256) {
			return fmt.Errorf("Checksum for %s does not match!", plugin.Id)
		}

		fmt.Printf("%s: Extracting\n", plugin.Id)
		err = exec.Command("tar", "-xf", downloadPath, "-C", downloadsDir).Run()
		if err != nil {
			return err
		}

		fmt.Printf("%s: Deleting archive\n", plugin.Id)
		err = exec.Command("rm", downloadPath).Run()
		if err != nil {
			return err
		}

		pluginPath := path.Join(downloadsDir, plugin.Id)

		pluginConfigBytes, err := ioutil.ReadFile(path.Join(pluginPath, "plugin.toml"))
		if err != nil {
			return err
		}

		var pluginConfig PluginConfig
		pluginConfigMd, err := toml.Decode(string(pluginConfigBytes), &pluginConfig)
		if err != nil {
			return err
		}

		// Default to only Linux support if plugin config was missing platforms
		if !pluginConfigMd.IsDefined("store", "platforms") {
			pluginConfig.Store.Platforms = platforms{
				Linux:   platform{true},
				Windows: platform{false},
				Darwin:     platform{false},
			}
		}

		outputPlugins[plugin.Id] = OutputPlugin{
			Id:                   plugin.Id,
			Name:                 pluginConfig.Name,
			Version:              plugin.Version,
			Archive:              plugin.Archive,
			Sha256:               plugin.Sha256,
			MinCrankshaftVersion: pluginConfig.MinCrankshaftVersion,

			Link:   pluginConfig.Link,
			Source: pluginConfig.Source,

			Author: author{
				Name: pluginConfig.Author.Name,
				Link: pluginConfig.Author.Link,
			},

			Store: store{
				Description: pluginConfig.Store.Description,
				Platforms:   pluginConfig.Store.Platforms,
			},
		}

		os.RemoveAll(pluginPath)
	}

	outputBytes, err := json.MarshalIndent(outputPlugins, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll("out", 0755); err != nil {
		return err
	}

	if err := os.WriteFile("out/plugins.json", outputBytes, 0755); err != nil {
		return err
	}

	return nil
}
