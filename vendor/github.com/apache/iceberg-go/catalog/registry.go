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

package catalog

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/config"
)

type registry map[string]Registrar

func (r registry) getKeys() []string {
	regMutex.Lock()
	defer regMutex.Unlock()

	return slices.Collect(maps.Keys(r))
}

func (r registry) set(catalogType string, reg Registrar) {
	regMutex.Lock()
	defer regMutex.Unlock()
	r[catalogType] = reg
}

func (r registry) get(catalogType string) (Registrar, bool) {
	regMutex.Lock()
	defer regMutex.Unlock()
	reg, ok := r[catalogType]

	return reg, ok
}

func (r registry) remove(catalogType string) {
	regMutex.Lock()
	defer regMutex.Unlock()
	delete(r, catalogType)
}

var (
	regMutex        sync.Mutex
	defaultRegistry = registry{}
)

// Registrar is a factory for creating Catalog instances, used for registering to use
// with LoadCatalog.
type Registrar interface {
	GetCatalog(ctx context.Context, catalogName string, props iceberg.Properties) (Catalog, error)
}

type RegistrarFunc func(context.Context, string, iceberg.Properties) (Catalog, error)

func (f RegistrarFunc) GetCatalog(ctx context.Context, catalogName string, props iceberg.Properties) (Catalog, error) {
	return f(ctx, catalogName, props)
}

// Register adds the new catalog type to the registry. If the catalog type is already registered, it will be replaced.
func Register(catalogType string, reg Registrar) {
	if reg == nil {
		panic("catalog: RegisterCatalog catalog factory is nil")
	}
	defaultRegistry.set(strings.ToLower(catalogType), reg)
}

// Unregister removes the requested catalog factory from the registry.
func Unregister(catalogType string) {
	defaultRegistry.remove(strings.ToLower(catalogType))
}

// GetRegisteredCatalogs returns the list of registered catalog names that can
// be looked up via LoadCatalog.
func GetRegisteredCatalogs() []string {
	return defaultRegistry.getKeys()
}

// Load allows loading a specific catalog by URI and properties.
//
// This is utilized alongside RegisterCatalog/UnregisterCatalog to not only allow
// easier catalog loading but also to allow for custom catalog implementations to
// be registered and loaded external to this module.
//
// The name parameter is used to look up the catalog configuration, if one exists,
// that was loaded from the configuration file, ".iceberg-go.yaml". By default,
// the config file is loaded from the user's home directory, but the directory can
// be changed by setting the GOICEBERG_HOME environment variable to the path of the
// directory containing the ".iceberg-go.yaml" file. The name will also be passed to
// the registered GetCatalog function.
//
// The catalog registry will be checked for the catalog's type. The "type" property
// is used first to search the catalog registry, with the passed properties taking
// priority over any loaded config.
//
// If there is no "type" in the configuration and no "type" in the passed in properties,
// then the "uri" property is used to look up the catalog by checking the scheme. Again,
// if there is an "uri" key set in the passed in "props" it will take priority over the
// configuration.
//
// Currently, the following catalog types are registered by default:
//
//   - "glue" for AWS Glue Data Catalog, the rest of the URI is ignored, all configuration
//     should be provided using the properties. "glue.region", "glue.endpoint",
//     "glue.max-retries", etc. Default AWS credentials are used if found, or can be
//     overridden by setting "glue.access-key-id", "glue.secret-access-key", and "glue.session-token".
//
//   - "rest" for a REST API catalog, if the properties have an "uri" key, then that will be used
//     as the REST endpoint, otherwise the URI is used as the endpoint. The REST catalog also
//     registers "http" and "https" so that Load with a http/s URI will automatically
//     load the REST Catalog.
func Load(ctx context.Context, name string, props iceberg.Properties) (Catalog, error) {
	if name == "" {
		name = config.EnvConfig.DefaultCatalog
	}

	conf := config.EnvConfig.Catalogs[name]
	if props == nil {
		props = iceberg.Properties{
			"uri":        conf.URI,
			"credential": conf.Credential,
			"warehouse":  conf.Warehouse,
		}
	} else {
		props = maps.Clone(props)
		props["uri"] = props.Get("uri", conf.URI)
		props["credential"] = props.Get("credential", conf.Credential)
		props["warehouse"] = props.Get("warehouse", conf.Warehouse)
	}

	catalogType := props.Get("type", conf.CatalogType)
	if catalogType == "" {
		if strings.Contains(props["uri"], "://") {
			uri, err := url.Parse(props["uri"])
			if err != nil {
				return nil, fmt.Errorf("failed to parse catalog URI: %w", err)
			}
			catalogType = uri.Scheme
		}
	}

	cat, ok := defaultRegistry.get(strings.ToLower(catalogType))
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrCatalogNotFound, catalogType)
	}

	return cat.GetCatalog(ctx, name, props)
}
