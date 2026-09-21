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
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestCredentialsDefaults(t *testing.T) {
	for name, credentials := range map[string]*Credentials{
		"nil":     nil,
		"empty":   {},
		"partial": {Username: ""},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, DefaultClusterUsername, credentials.User())
			require.Equal(t, DefaultClusterPassword, credentials.Pass())
			require.Equal(t, "Administrator:asdasd", credentials.UserPass())
		})
	}
}

func TestCredentialsProvided(t *testing.T) {
	credentials := &Credentials{Username: "username", Password: "password"}

	require.Equal(t, "username", credentials.User())
	require.Equal(t, "password", credentials.Pass())
	require.Equal(t, "username:password", credentials.UserPass())
}

func TestCredentialsQuoting(t *testing.T) {
	type test struct {
		name     string
		password string
		expected string
	}

	tests := []*test{
		{
			name:     "plain",
			password: "password",
			expected: `'password'`,
		},
		{
			name:     "shell expansion",
			password: "pass$(whoami)word",
			expected: `'pass$(whoami)word'`,
		},
		{
			name:     "double quotes",
			password: `pass"word`,
			expected: `'pass"word'`,
		},
		{
			name:     "single quote",
			password: "pass'word",
			expected: `'pass'\''word'`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			credentials := &Credentials{Username: "username", Password: test.password}

			require.Equal(t, `'username'`, credentials.QuotedUser())
			require.Equal(t, test.expected, credentials.QuotedPass())
		})
	}
}

func TestCredentialsUnmarshal(t *testing.T) {
	type overlay struct {
		Credentials *Credentials `yaml:"credentials,omitempty"`
	}

	var decoded overlay

	require.NoError(t, yaml.Unmarshal([]byte("credentials:\n  username: username\n  password: password"), &decoded))
	require.Equal(t, &Credentials{Username: "username", Password: "password"}, decoded.Credentials)
}

func TestCredentialsMarshalRedacted(t *testing.T) {
	credentials := &Credentials{Username: "username", Password: "password"}

	asJSON, err := json.Marshal(credentials)
	require.NoError(t, err)
	require.JSONEq(t, `"<redacted>"`, string(asJSON))

	asYAML, err := yaml.Marshal(credentials)
	require.NoError(t, err)
	require.Equal(t, "<redacted>\n", string(asYAML))

	// The credentials may be nested inside a structure which is itself marshaled, which must be redacted in the same
	// way rather than silently including them.
	asJSON, err = json.Marshal(struct {
		Credentials *Credentials `json:"credentials"`
	}{Credentials: credentials})
	require.NoError(t, err)
	require.JSONEq(t, `{"credentials":"<redacted>"}`, string(asJSON))

	require.Equal(t, "<redacted>", credentials.String())

	// The password shouldn't be leaked when the credentials are printed as part of a surrounding structure either.
	printed := fmt.Sprintf("%v", struct{ Credentials *Credentials }{Credentials: credentials})
	require.NotContains(t, printed, "password")
}

func TestBucketName(t *testing.T) {
	var nilBlueprint *BucketBlueprint

	require.Equal(t, DefaultBucketName, nilBlueprint.BucketName())
	require.Equal(t, DefaultBucketName, (&BucketBlueprint{}).BucketName())
	require.Equal(t, "bucket", (&BucketBlueprint{Name: "bucket"}).BucketName())
}
