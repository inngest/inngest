// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	cfgFile           = ".iceberg-go.yaml"
	defaultMaxWorkers = 5
)

type Config struct {
	DefaultCatalog string                   `yaml:"default-catalog"`
	Catalogs       map[string]CatalogConfig `yaml:"catalog"`
	MaxWorkers     int                      `yaml:"max-workers"`
}

type RestOptions struct {
	SigV4Enabled  bool   `yaml:"sigv4-enabled"`
	SigningName   string `yaml:"signing-name"`
	SigningRegion string `yaml:"signing-region"`
}

type CatalogConfig struct {
	CatalogType string       `yaml:"type"`
	URI         string       `yaml:"uri"`
	Output      string       `yaml:"output"`
	Credential  string       `yaml:"credential"`
	Warehouse   string       `yaml:"warehouse"`
	AwsProfile  string       `yaml:"aws-profile"`
	SQLDriver   string       `yaml:"sql-driver"`
	SQLDialect  string       `yaml:"sql-dialect"`
	RestOptions *RestOptions `yaml:"rest,omitempty"`
}

// LoadConfig returns configuration data when it can be read. It retains the
// historical best-effort behavior; callers that must distinguish a missing
// default from a broken explicit file should use LoadConfigFile.
func LoadConfig(configPath string) []byte {
	data, _ := LoadConfigFile(configPath)

	return data
}

// LoadConfigFile loads configuration data and reports path and read failures.
// A missing implicit home-directory configuration is not an error.
func LoadConfigFile(configPath string) ([]byte, error) {
	implicit := configPath == ""
	var path string
	if !implicit {
		path = configPath
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home directory for config: %w", err)
		}
		path = filepath.Join(homeDir, cfgFile)
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path %q: %w", path, err)
	}

	file, err := os.ReadFile(path)
	if err != nil {
		if implicit && errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	return file, nil
}

// ParseConfig unmarshals the config file once and resolves the catalog to use.
// When catalogName is empty, it falls back to the file's default-catalog field and
// then to the built-in "default" name. It returns the matching CatalogConfig, or
// nil/nil if no such catalog is defined in the file. A non-nil error means the
// file could not be parsed and should be surfaced to the user.
func ParseConfig(file []byte, catalogName string) (*CatalogConfig, error) {
	if len(file) == 0 {
		return nil, nil
	}
	var config Config
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	if catalogName == "" {
		catalogName = config.DefaultCatalog
	}
	if catalogName == "" {
		catalogName = "default"
	}

	res, ok := config.Catalogs[catalogName]
	if !ok {
		return nil, nil
	}

	return &res, nil
}

func fromConfigFiles() Config {
	dir := os.Getenv("GOICEBERG_HOME")
	if dir != "" {
		dir = filepath.Join(dir, cfgFile)
	}

	var cfg Config
	if err := yaml.Unmarshal(LoadConfig(dir), &cfg); err != nil {
		return cfg
	}

	if cfg.DefaultCatalog == "" {
		cfg.DefaultCatalog = "default"
	}
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = defaultMaxWorkers
	}

	return cfg
}

var EnvConfig = fromConfigFiles()
