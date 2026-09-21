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

package value

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// DefaultClusterUsername is the username used when one hasn't been provided in the config; it matches the username
	// used by 'cluster_run'.
	DefaultClusterUsername = "Administrator"

	// DefaultClusterPassword is the password used when one hasn't been provided in the config; it matches the password
	// used by 'cluster_run'.
	DefaultClusterPassword = "asdasd"
)

// Credentials encapsulates the administrator credentials for a Couchbase Cluster. These only need to be provided when
// running against a cluster which wasn't provisioned by 'cbtools-autobench'.
//
// NOTE: The accessors below may all be called on a nil receiver, in which case the defaults are returned.
type Credentials struct {
	Username string `json:"username" yaml:"username,omitempty"`
	Password string `json:"password" yaml:"password,omitempty"`
}

// MarshalJSON returns a redacted representation of the credentials; they're unmarshaled from the config only, so
// marshaling them must not leak the password into a report/log.
func (c *Credentials) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

// MarshalYAML returns a redacted representation of the credentials; they're unmarshaled from the config only, so
// marshaling them must not leak the password back out to disk.
//
// NOTE: The error return is dictated by the 'yaml.Marshaler' interface.
func (c *Credentials) MarshalYAML() (any, error) { //nolint:unparam
	return c.String(), nil
}

// String returns a redacted representation of the credentials, meaning the password isn't leaked when the credentials
// are interpolated into a string or a log field.
//
// NOTE: The username/password may still be printed via the accessors, which is what the commands run on the remote
// machines require.
func (c *Credentials) String() string {
	return "<redacted>"
}

// User returns the username which should be used when connecting to the cluster.
func (c *Credentials) User() string {
	if c == nil || c.Username == "" {
		return DefaultClusterUsername
	}

	return c.Username
}

// Pass returns the password which should be used when connecting to the cluster.
func (c *Credentials) Pass() string {
	if c == nil || c.Password == "" {
		return DefaultClusterPassword
	}

	return c.Password
}

// UserPass returns the credentials in the 'username:password' format accepted by 'curl'.
func (c *Credentials) UserPass() string {
	return fmt.Sprintf("%s:%s", c.User(), c.Pass())
}

// QuotedUser returns the username, quoted for use in a command which will be run in a remote shell.
func (c *Credentials) QuotedUser() string {
	return quote(c.User())
}

// QuotedPass returns the password, quoted for use in a command which will be run in a remote shell.
func (c *Credentials) QuotedPass() string {
	return quote(c.Pass())
}

// QuotedUserPass returns the credentials in the 'username:password' format accepted by 'curl', quoted for use in a
// command which will be run in a remote shell.
func (c *Credentials) QuotedUserPass() string {
	return quote(c.UserPass())
}

// quote single quotes the given string, meaning any characters which are special to the shell are passed through
// verbatim; this matters because credentials are user provided and commonly contain such characters.
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
