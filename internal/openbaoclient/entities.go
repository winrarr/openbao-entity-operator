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
)

const (
	identityPathSegment = "identity"
	entityPathSegment   = "entity"
)

// Entity is the OpenBao identity entity surface reconciled by this project.
type Entity struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
	Policies []string          `json:"policies"`
	Disabled bool              `json:"disabled"`
}

// EntityRequest is the mutable OpenBao entity configuration.
type EntityRequest struct {
	Name     string            `json:"name,omitempty"`
	Metadata map[string]string `json:"metadata"`
	Policies []string          `json:"policies"`
	Disabled bool              `json:"disabled"`
}

// CreateEntity creates an entity and returns its generated identifier.
func (c *Client) CreateEntity(ctx context.Context, request EntityRequest) (string, error) {
	var response apiResponse[struct {
		ID string `json:"id"`
	}]
	if err := c.do(ctx, http.MethodPost, identityPathSegment+"/"+entityPathSegment, request, &response); err != nil {
		return "", err
	}
	return response.Data.ID, nil
}

// GetEntityByID reads an entity by its stable identifier.
func (c *Client) GetEntityByID(ctx context.Context, id string) (*Entity, error) {
	var response apiResponse[Entity]
	if err := c.doSegments(ctx, http.MethodGet, []string{identityPathSegment, entityPathSegment, "id", id}, nil, &response); err != nil {
		return nil, err
	}
	response.Data.ID = firstNonEmpty(response.Data.ID, id)
	return &response.Data, nil
}

// GetEntityByName reads an entity by its OpenBao name.
func (c *Client) GetEntityByName(ctx context.Context, name string) (*Entity, error) {
	var response apiResponse[Entity]
	if err := c.doSegments(ctx, http.MethodGet, []string{identityPathSegment, entityPathSegment, "name", name}, nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// UpdateEntity updates an entity by ID and returns the new representation.
func (c *Client) UpdateEntity(ctx context.Context, id string, request EntityRequest) (*Entity, error) {
	var response apiResponse[Entity]
	if err := c.doSegments(ctx, http.MethodPost, []string{identityPathSegment, entityPathSegment, "id", id}, request, &response); err != nil {
		return nil, err
	}
	response.Data.ID = firstNonEmpty(response.Data.ID, id)
	return &response.Data, nil
}

// DeleteEntity deletes an entity by ID.
func (c *Client) DeleteEntity(ctx context.Context, id string) error {
	return c.doSegments(ctx, http.MethodDelete, []string{identityPathSegment, entityPathSegment, "id", id}, nil, nil)
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
