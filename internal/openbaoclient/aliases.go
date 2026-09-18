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

const entityAliasPathSegment = "entity-alias"

// EntityAlias is the OpenBao identity entity alias surface reconciled by this project.
type EntityAlias struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	MountAccessor string `json:"mount_accessor"`
	CanonicalID   string `json:"canonical_id"`
	EntityID      string `json:"entity_id"`
}

// EntityAliasRequest is the OpenBao entity alias configuration.
type EntityAliasRequest struct {
	Name          string `json:"name,omitempty"`
	MountAccessor string `json:"mount_accessor,omitempty"`
	CanonicalID   string `json:"canonical_id,omitempty"`
}

// CreateEntityAlias creates an alias and returns its generated identifier.
func (c *Client) CreateEntityAlias(ctx context.Context, request EntityAliasRequest) (string, error) {
	var response apiResponse[struct {
		ID string `json:"id"`
	}]
	if err := c.do(ctx, http.MethodPost, identityPathSegment+"/"+entityAliasPathSegment, request, &response); err != nil {
		return "", err
	}
	return response.Data.ID, nil
}

// ListEntityAliasIDs lists the stable identifiers of all OpenBao entity aliases.
func (c *Client) ListEntityAliasIDs(ctx context.Context) ([]string, error) {
	var response apiResponse[struct {
		Keys []string `json:"keys"`
	}]
	if err := c.doSegmentsQuery(ctx, http.MethodGet, []string{identityPathSegment, entityAliasPathSegment, "id"}, url.Values{listQueryKey: {trueQueryValue}}, nil, &response); err != nil {
		return nil, err
	}
	return response.Data.Keys, nil
}

// GetEntityAliasByID reads an alias by its stable identifier.
func (c *Client) GetEntityAliasByID(ctx context.Context, id string) (*EntityAlias, error) {
	var response apiResponse[EntityAlias]
	if err := c.doSegments(ctx, http.MethodGet, []string{identityPathSegment, entityAliasPathSegment, "id", id}, nil, &response); err != nil {
		return nil, err
	}
	response.Data.ID = firstNonEmpty(response.Data.ID, id)
	response.Data.CanonicalID = firstNonEmpty(response.Data.CanonicalID, response.Data.EntityID)
	return &response.Data, nil
}

// UpdateEntityAlias updates an alias by ID and returns the new representation.
func (c *Client) UpdateEntityAlias(ctx context.Context, id string, request EntityAliasRequest) (*EntityAlias, error) {
	var response apiResponse[EntityAlias]
	if err := c.doSegments(ctx, http.MethodPost, []string{identityPathSegment, entityAliasPathSegment, "id", id}, request, &response); err != nil {
		return nil, err
	}
	response.Data.ID = firstNonEmpty(response.Data.ID, id)
	response.Data.CanonicalID = firstNonEmpty(response.Data.CanonicalID, response.Data.EntityID)
	return &response.Data, nil
}

// DeleteEntityAlias deletes an alias by ID.
func (c *Client) DeleteEntityAlias(ctx context.Context, id string) error {
	return c.doSegments(ctx, http.MethodDelete, []string{identityPathSegment, entityAliasPathSegment, "id", id}, nil, nil)
}
