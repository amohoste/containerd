// +build linux

/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package devmapper

import (
	"fmt"
	"os"

	"github.com/docker/go-units"
	"github.com/hashicorp/go-multierror"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

// Config represents device mapper configuration loaded from file.
// Size units can be specified in human-readable string format (like "32KIB", "32GB", "32Tb")
type Config struct {
	// Device snapshotter root directory for metadata
	RootPath string `toml:"root_path"`

	// Name for 'thin-pool' device to be used by snapshotter (without /dev/mapper/ prefix)
	PoolName string `toml:"pool_name"`

	// Defines how much space to allocate when creating base image for container
	BaseImageSize      string `toml:"base_image_size"`
	BaseImageSizeBytes uint64 `toml:"-"`

	// Flag to async remove device using Cleanup() callback in snapshots GC
	AsyncRemove bool `toml:"async_remove"`

	// Whether to discard blocks when removing a thin device.
	DiscardBlocks bool `toml:"discard_blocks"`
}

// LoadConfig reads devmapper configuration file from disk in TOML format
func LoadConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}

		return nil, err
	}

	config := Config{}
	file, err := toml.LoadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open devmapepr TOML: %s", path)
	}

	if err := file.Unmarshal(&config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal devmapper TOML")
	}

	if err := config.parse(); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) parse() error {
	baseImageSize, err := units.RAMInBytes(c.BaseImageSize)
	if err != nil {
		return errors.Wrapf(err, "failed to parse base image size: '%s'", c.BaseImageSize)
	}

	c.BaseImageSizeBytes = uint64(baseImageSize)
	return nil
}

// Validate makes sure configuration fields are valid
func (c *Config) Validate() error {
	var result *multierror.Error

	if c.PoolName == "" {
		result = multierror.Append(result, fmt.Errorf("pool_name is required"))
	}

	if c.RootPath == "" {
		result = multierror.Append(result, fmt.Errorf("root_path is required"))
	}

	if c.BaseImageSize == "" {
		result = multierror.Append(result, fmt.Errorf("base_image_size is required"))
	}

	return result.ErrorOrNil()
}
