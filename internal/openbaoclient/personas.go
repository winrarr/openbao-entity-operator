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

const personaPathSegment = "persona"

// Persona is the durable OpenBao identity persona representation.
type Persona struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	EntityID      string            `json:"entity_id"`
	MountAccessor string            `json:"mount_accessor"`
	Metadata      map[string]string `json:"metadata"`
}

// PersonaRequest is the mutable OpenBao persona configuration.
type PersonaRequest struct {
	Name          string            `json:"name,omitempty"`
	EntityID      string            `json:"entity_id,omitempty"`
	MountAccessor string            `json:"mount_accessor,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// CreatePersona creates a persona and returns its generated identifier.
func (c *Client) CreatePersona(ctx context.Context, request PersonaRequest) (string, error) {
	var response apiResponse[struct {
		ID string `json:"id"`
	}]
	if err := c.do(ctx, http.MethodPost, "identity/persona", request, &response); err != nil {
		return "", err
	}
	return response.Data.ID, nil
}

// ListPersonaIDs lists stable persona identifiers.
func (c *Client) ListPersonaIDs(ctx context.Context) ([]string, error) {
	var response apiResponse[struct {
		Keys []string `json:"keys"`
	}]
	if err := c.doSegmentsQuery(ctx, http.MethodGet, []string{identityPathSegment, personaPathSegment, "id"}, url.Values{listQueryKey: {trueQueryValue}}, nil, &response); err != nil {
		return nil, err
	}
	return response.Data.Keys, nil
}

// GetPersonaByID reads a persona by stable identifier.
func (c *Client) GetPersonaByID(ctx context.Context, id string) (*Persona, error) {
	var response apiResponse[Persona]
	if err := c.doSegments(ctx, http.MethodGet, []string{identityPathSegment, personaPathSegment, "id", id}, nil, &response); err != nil {
		return nil, err
	}
	response.Data.ID = firstNonEmpty(response.Data.ID, id)
	return &response.Data, nil
}

// UpdatePersona updates a persona by stable identifier.
func (c *Client) UpdatePersona(ctx context.Context, id string, request PersonaRequest) (*Persona, error) {
	var response apiResponse[Persona]
	if err := c.doSegments(ctx, http.MethodPost, []string{identityPathSegment, personaPathSegment, "id", id}, request, &response); err != nil {
		return nil, err
	}
	response.Data.ID = firstNonEmpty(response.Data.ID, id)
	return &response.Data, nil
}

// DeletePersona deletes a persona by stable identifier.
func (c *Client) DeletePersona(ctx context.Context, id string) error {
	return c.doSegments(ctx, http.MethodDelete, []string{identityPathSegment, personaPathSegment, "id", id}, nil, nil)
}
