/*
Copyright 2026 openbao-entity-operator contributors.

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

package openbao

import (
	"reflect"
	"testing"
)

func TestParseWatchNamespaces(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "empty means all namespaces", raw: "", want: nil},
		{name: "whitespace means all namespaces", raw: "  ", want: nil},
		{name: "trims and sorts", raw: "team-b, team-a", want: []string{"team-a", "team-b"}},
		{name: "rejects empty entry", raw: "team-a,,team-b", wantErr: true},
		{name: "rejects duplicate", raw: "team-a,team-a", wantErr: true},
		{name: "rejects invalid name", raw: "Team-A", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseWatchNamespaces(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseWatchNamespaces(%q) error = %v, wantErr %t", tt.raw, err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseWatchNamespaces(%q) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestCacheOptionsForWatchNamespaces(t *testing.T) {
	options, err := CacheOptionsForWatchNamespaces("team-a,team-b")
	if err != nil {
		t.Fatalf("CacheOptionsForWatchNamespaces() error = %v", err)
	}
	if len(options.DefaultNamespaces) != 2 {
		t.Fatalf("DefaultNamespaces length = %d, want 2", len(options.DefaultNamespaces))
	}
	for _, namespace := range []string{"team-a", "team-b"} {
		if _, ok := options.DefaultNamespaces[namespace]; !ok {
			t.Errorf("DefaultNamespaces is missing %q", namespace)
		}
	}

	unscoped, err := CacheOptionsForWatchNamespaces("")
	if err != nil {
		t.Fatalf("CacheOptionsForWatchNamespaces(\"\") error = %v", err)
	}
	if unscoped.DefaultNamespaces != nil {
		t.Fatalf("empty scope DefaultNamespaces = %#v, want nil", unscoped.DefaultNamespaces)
	}
}
