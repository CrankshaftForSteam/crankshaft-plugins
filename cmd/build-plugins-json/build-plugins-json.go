package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
)

type InputPlugin struct {
	Id               string `json:"id"`
	Repo             string `json:"repo"`
	PluginConfigPath string `json:"pluginConfigPath"`

	Version string `json:"version"`
	Archive string `json:"archive"`
	Sha256  string `json:"sha256"`

	Source string `json:"source"`
}

type InputFile struct {
	Plugins []InputPlugin `json:"plugins"`
}

const INPUT_FILE = "plugins.json"

type author struct {
	Name string `toml:"name" json:"name"`
	Link string `toml:"link" json:"link"`
}

type store struct {
	Description string `toml:"description" json:"description"`
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

	for _, plugin := range inputPlugins {

		// TODO: only checkout plugin.toml
		// couldn't figure out how to checkout one file after cloning with NoCheckout
		// (could probably just use git tbh)

		fs := memfs.New()

		cloneUrl := plugin.Repo
		if cloneUrl == "" {
			cloneUrl = plugin.Source
		}

		fmt.Printf("Cloning %s from %s\n", plugin.Id, cloneUrl)

		_, err = git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
			URL: cloneUrl,
			// NoCheckout: true,
			Depth: 1,
		})
		if err != nil {
			return err
		}

		pluginConfigPath := plugin.PluginConfigPath
		if pluginConfigPath == "" {
			pluginConfigPath = "plugin.toml"
		}

		pluginConfigFile, err := fs.Open(pluginConfigPath)
		if err != nil {
			return err
		}

		buf := new(strings.Builder)
		_, err = io.Copy(buf, pluginConfigFile)
		if err != nil {
			return err
		}
		pluginConfigString := buf.String()

		var pluginConfig PluginConfig
		if _, err := toml.Decode(pluginConfigString, &pluginConfig); err != nil {
			return err
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
			},
		}
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
