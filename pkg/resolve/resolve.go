// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resolve

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dprotaso/go-yit"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/publish"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// ImageReferences resolves supported references to images within the input yaml
// to published image digests.
//
// If a reference can be built and pushed, its yaml.Node will be mutated.
func ImageReferences(ctx context.Context, docs []*yaml.Node, builder build.Interface, publisher publish.Interface) error {
	// First, walk the input objects and collect a list of supported references
	refs := make(map[string][]*yaml.Node)
	configRefs := make(map[string][]*yaml.Node)

	for _, doc := range docs {
		it := refsFromDoc(doc)

		for node, ok := it(); ok; node, ok = it() {
			ref := strings.TrimSpace(node.Value)

			if err := builder.IsSupportedReference(ref); err != nil {

				// TODO(jdolitsky): further validation of config ref to produce other errors.
				// Currently, if this error is returned, it is silently ignored.
				// At this point, it means that either a) the ko:// ref was invalid
				// or b) the ref is simply not prefixed with koconfig://
				if configErr := builder.IsSupportedConfigReference(ref); configErr == nil {
					configRefs[ref] = append(configRefs[ref], node)
					continue
				}

				return fmt.Errorf("found strict reference but %s is not a valid import path: %w", ref, err)
			}

			refs[ref] = append(refs[ref], node)
		}
	}

	// Next, perform parallel builds for each of the supported references.
	var sm sync.Map
	var errg errgroup.Group
	for ref := range refs {
		ref := ref
		errg.Go(func() error {
			img, err := builder.Build(ctx, ref)
			if err != nil {
				return err
			}
			digest, err := publisher.Publish(ctx, img, ref)
			if err != nil {
				return err
			}
			sm.Store(ref, digest.String())
			return nil
		})
	}
	if err := errg.Wait(); err != nil {
		return err
	}

	// Walk the tags and update them with their digest.
	for ref, nodes := range refs {
		digest, ok := sm.Load(ref)

		if !ok {
			return fmt.Errorf("resolved reference to %q not found", ref)
		}

		for _, node := range nodes {
			node.Value = digest.(string)
		}
	}

	// Finally, inject any config references with the proper value
	var overrides map[interface{}]interface{}
	if v := ctx.Value(build.StrictConfigScheme); v != nil {
		overrides = v.(map[interface{}]interface{})
	}
	for configRef, nodes := range configRefs {
		value := lookupConfigValue(configRef, overrides)
		for _, node := range nodes {
			node.Value = value
		}
	}

	return nil
}

// This currently returns anything prefixed with ko:// or koconfig://
func refsFromDoc(doc *yaml.Node) yit.Iterator {
	it := yit.FromNode(doc).
		RecurseNodes().
		Filter(yit.StringValue)

	return it.Filter(yit.Union(
		yit.WithPrefix(build.StrictScheme),
		yit.WithPrefix(build.StrictConfigScheme)))
}

// Attempt to locate a config value (from a key in the form "x.y.z")
// from a nested override map. If the key is not found, the default
// value is returned. If no default is provided, return empty string.
//
// A default value is defined by everything after the first forward
// slash character ("/") in the config reference.
//
// Example: koconfig://my.nested.key/my-default (default: "my-default")
func lookupConfigValue(configRef string, overrides map[interface{}]interface{}) string {
	configRef = strings.TrimPrefix(configRef, build.StrictConfigScheme)
	parts := strings.Split(configRef, "/")
	key := parts[0]
	value := strings.Join(parts[1:], "/")
	keyParts := strings.Split(key, ".")
	lastIndex := len(keyParts) - 1
	child := overrides
	for i, keyPart := range keyParts {
		v, ok := child[keyPart]
		if !ok {
			break
		}
		if i == lastIndex {
			value = fmt.Sprintf("%v", v)
			break
		}
		child = v.(map[interface{}]interface{})
	}
	return value
}
