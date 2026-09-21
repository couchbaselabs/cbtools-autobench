// Copyright 2021 Couchbase Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"

	"github.com/couchbaselabs/cbtools-autobench/nodes"

	"github.com/couchbase/tools-common/sync/v2/hofp"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// provisionOptions encapsulates the possible options which can be used to change the behavior of the 'provision'
// sub-command.
var provisionOptions = struct {
	configPath string

	// loadOnly skips actual provisioning i.e. just flush and load the test dataset; this is useful when benchmarking
	// multiple datasets whilst using the same cluster.
	loadOnly bool
}{}

// provisionCommand is the provision sub-command, used to provision a cluster and load a test dataset.
var provisionCommand = &cobra.Command{
	RunE:  provision,
	Short: "install and configure a cluster and backup client",
	Use:   "provision",
}

// init the flags/arguments for the provision sub-command.
func init() {
	provisionCommand.Flags().StringVarP(
		&provisionOptions.configPath,
		"config",
		"c",
		"",
		"path to a cbtools-autobench config file",
	)

	provisionCommand.Flags().BoolVarP(
		&provisionOptions.loadOnly,
		"load-only",
		"",
		false,
		"skip provisioning and only load benchmark dataset",
	)

	markFlagRequired(provisionCommand, "config")
}

// provision sub-command, this will use the provided configuration to provision a cluster/backup client and load a test
// dataset.
func provision(_ *cobra.Command, _ []string) error {
	config, err := readConfig(provisionOptions.configPath)
	if err != nil {
		return errors.Wrap(err, "failed to read autobench config")
	}

	cluster, err := nodes.NewCluster(config.SSHConfig, config.Blueprint.Cluster)
	if err != nil {
		return errors.Wrap(err, "failed to connect to cluster")
	}
	defer cluster.Close()

	type provisioner interface {
		Provision() error
	}

	provisioners := []provisioner{cluster}

	if config.Blueprint.BackupClient == nil && !provisionOptions.loadOnly {
		return errors.New("a backup client must be provided when provisioning, see the '--load-only' flag")
	}

	// The backup client is only required when provisioning; when we're loading a dataset into an existing cluster it
	// may be omitted from the config entirely.
	if config.Blueprint.BackupClient != nil {
		client, err := nodes.NewBackupClient(config.SSHConfig, config.Blueprint.BackupClient)
		if err != nil {
			return errors.Wrap(err, "failed to connect to backup client")
		}
		defer client.Close()

		provisioners = append(provisioners, client)
	}

	if provisionOptions.loadOnly {
		provisioners = nil
	}

	pool := hofp.NewPool(hofp.Options{Size: 2})

	queue := func(p provisioner) error {
		return pool.Queue(func(_ context.Context) error { return p.Provision() })
	}

	for _, p := range provisioners {
		if queue(p) != nil {
			break
		}
	}

	err = pool.Stop()
	if err != nil {
		return errors.Wrap(err, "unexpected error whilst provisioning")
	}

	err = cluster.LoadData(config.Blueprint.Cluster.Bucket.Compact)
	if err != nil {
		return errors.Wrap(err, "failed to load test dataset")
	}

	return nil
}
