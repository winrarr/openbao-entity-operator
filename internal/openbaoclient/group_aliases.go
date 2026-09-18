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

package openbaoclient

import (
	"context"
	"net/http"
	"net/url"
)

const groupAliasPathSegment = "group-alias"

// GroupAlias is the OpenBao identity group alias surface reconciled by this project.
type GroupAlias struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	MountAccessor string `json:"mount_accessor"`
	CanonicalID   string `json:"canonical_id"`
}

// GroupAliasRequest is the OpenBao group alias configuration.
type GroupAliasRequest struct {
	Name          string `json:"name,omitempty"`
	MountAccessor string `json:"mount_accessor,omitempty"`
	CanonicalID   string `json:"canonical_id,omitempty"`
}

// CreateGroupAlias creates an alias and returns its generated identifier.
func (c *Client) CreateGroupAlias(ctx context.Context, request GroupAliasRequest) (string, error) {
	var response apiResponse[struct {
		ID string `json:"id"`
	}]
	if err := c.do(ctx, http.MethodPost, identityPathSegment+"/"+groupAliasPathSegment, request, &response); err != nil {
		return "", err
	}
	return response.Data.ID, nil
}

// ListGroupAliasIDs lists the stable identifiers of all OpenBao group aliases.
func (c *Client) ListGroupAliasIDs(ctx context.Context) ([]string, error) {
	var response apiResponse[struct {
		Keys []string `json:"keys"`
	}]
	if err := c.doSegmentsQuery(ctx, http.MethodGet, []string{identityPathSegment, groupAliasPathSegment, "id"}, url.Values{listQueryKey: {trueQueryValue}}, nil, &response); err != nil {
		return nil, err
	}
	return response.Data.Keys, nil
}

// GetGroupAliasByID reads an alias by its stable identifier.
func (c *Client) GetGroupAliasByID(ctx context.Context, id string) (*GroupAlias, error) {
	var response apiResponse[GroupAlias]
	if err := c.doSegments(ctx, http.MethodGet, []string{identityPathSegment, groupAliasPathSegment, "id", id}, nil, &response); err != nil {
		return nil, err
	}
	response.Data.ID = firstNonEmpty(response.Data.ID, id)
	return &response.Data, nil
}

// UpdateGroupAlias updates an alias by ID and returns the new representation.
func (c *Client) UpdateGroupAlias(ctx context.Context, id string, request GroupAliasRequest) (*GroupAlias, error) {
	var response apiResponse[GroupAlias]
	if err := c.doSegments(ctx, http.MethodPost, []string{identityPathSegment, groupAliasPathSegment, "id", id}, request, &response); err != nil {
		return nil, err
	}
	response.Data.ID = firstNonEmpty(response.Data.ID, id)
	return &response.Data, nil
}

// DeleteGroupAlias deletes an alias by ID.
func (c *Client) DeleteGroupAlias(ctx context.Context, id string) error {
	return c.doSegments(ctx, http.MethodDelete, []string{identityPathSegment, groupAliasPathSegment, "id", id}, nil, nil)
}
