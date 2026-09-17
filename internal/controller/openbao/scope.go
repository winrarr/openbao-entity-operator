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
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

// CacheOptionsForWatchNamespaces converts the operator's comma-separated
// namespace flag into controller-runtime cache options. An empty value keeps
// the historical cluster-wide cache behavior.
func CacheOptionsForWatchNamespaces(raw string) (cache.Options, error) {
	namespaces, err := ParseWatchNamespaces(raw)
	if err != nil {
		return cache.Options{}, err
	}
	if len(namespaces) == 0 {
		return cache.Options{}, nil
	}

	defaultNamespaces := make(map[string]cache.Config, len(namespaces))
	for _, namespace := range namespaces {
		defaultNamespaces[namespace] = cache.Config{}
	}
	return cache.Options{DefaultNamespaces: defaultNamespaces}, nil
}

// ParseWatchNamespaces validates and normalizes a comma-separated namespace
// allowlist. Empty input means that the manager watches all namespaces.
func ParseWatchNamespaces(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	seen := make(map[string]struct{})
	namespaces := make([]string, 0)
	for value := range strings.SplitSeq(raw, ",") {
		namespace := strings.TrimSpace(value)
		if namespace == "" {
			return nil, fmt.Errorf("watch namespace list contains an empty entry")
		}
		if messages := validation.IsDNS1123Label(namespace); len(messages) > 0 {
			return nil, fmt.Errorf("invalid watch namespace %q: %s", namespace, strings.Join(messages, "; "))
		}
		if _, ok := seen[namespace]; ok {
			return nil, fmt.Errorf("watch namespace %q is listed more than once", namespace)
		}
		seen[namespace] = struct{}{}
		namespaces = append(namespaces, namespace)
	}

	slices.Sort(namespaces)
	return namespaces, nil
}
