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

const groupPathSegment = "group"

// Group is the OpenBao identity group surface reconciled by this project.
type Group struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Type            string            `json:"type"`
	Metadata        map[string]string `json:"metadata"`
	Policies        []string          `json:"policies"`
	MemberEntityIDs []string          `json:"member_entity_ids"`
	MemberGroupIDs  []string          `json:"member_group_ids"`
}

// GroupRequest is the mutable OpenBao identity group configuration.
type GroupRequest struct {
	Name            string            `json:"name,omitempty"`
	Type            string            `json:"type,omitempty"`
	Metadata        map[string]string `json:"metadata"`
	Policies        []string          `json:"policies"`
	MemberEntityIDs []string          `json:"member_entity_ids"`
	MemberGroupIDs  []string          `json:"member_group_ids"`
}

// CreateGroup creates a group and returns its generated identifier.
func (c *Client) CreateGroup(ctx context.Context, request GroupRequest) (string, error) {
	var response apiResponse[struct {
		ID string `json:"id"`
	}]
	if err := c.do(ctx, http.MethodPost, identityPathSegment+"/"+groupPathSegment, request, &response); err != nil {
		return "", err
	}
	return response.Data.ID, nil
}

// GetGroupByID reads a group by its stable identifier.
func (c *Client) GetGroupByID(ctx context.Context, id string) (*Group, error) {
	var response apiResponse[Group]
	if err := c.doSegments(ctx, http.MethodGet, []string{identityPathSegment, groupPathSegment, "id", id}, nil, &response); err != nil {
		return nil, err
	}
	response.Data.ID = firstNonEmpty(response.Data.ID, id)
	return &response.Data, nil
}

// GetGroupByName reads a group by its OpenBao name.
func (c *Client) GetGroupByName(ctx context.Context, name string) (*Group, error) {
	var response apiResponse[Group]
	if err := c.doSegments(ctx, http.MethodGet, []string{identityPathSegment, groupPathSegment, "name", name}, nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// UpdateGroup updates a group by ID and returns the response representation.
func (c *Client) UpdateGroup(ctx context.Context, id string, request GroupRequest) (*Group, error) {
	var response apiResponse[Group]
	if err := c.doSegments(ctx, http.MethodPost, []string{identityPathSegment, groupPathSegment, "id", id}, request, &response); err != nil {
		return nil, err
	}
	response.Data.ID = firstNonEmpty(response.Data.ID, id)
	return &response.Data, nil
}

// DeleteGroup deletes a group by ID.
func (c *Client) DeleteGroup(ctx context.Context, id string) error {
	return c.doSegments(ctx, http.MethodDelete, []string{identityPathSegment, groupPathSegment, "id", id}, nil, nil)
}
