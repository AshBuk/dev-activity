// Copyright (c) 2026 Asher Buk
// SPDX-License-Identifier: MIT

// Package github fetches user activity from the GitHub REST API.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiURL  = "https://api.github.com"
	perPage = 100
	// The events API keeps at most 300 events / 90 days of history,
	// so 3 pages of 100 is everything GitHub will ever give us.
	maxPages = 3
)

// Event is the subset of a GitHub event we care about. Payload is kept
// raw (jsonb in the database) so new queries never require code changes.
type Event struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Repo struct {
		Name string `json:"name"`
	} `json:"repo"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type Client struct {
	http  *http.Client
	token string
}

// NewClient returns a client; token may be empty for public data only.
func NewClient(token string) *Client {
	return &Client{
		http:  &http.Client{Timeout: 30 * time.Second},
		token: token,
	}
}

// Events returns the user's recent activity, newest first.
func (c *Client) Events(ctx context.Context, user string) ([]Event, error) {
	var all []Event
	for page := 1; page <= maxPages; page++ {
		url := fmt.Sprintf("%s/users/%s/events?per_page=%d&page=%d", apiURL, user, perPage, page)
		events, err := c.fetchPage(ctx, url)
		if err != nil {
			return nil, err
		}
		all = append(all, events...)
		if len(events) < perPage {
			break // last page
		}
	}
	return all, nil
}

func (c *Client) fetchPage(ctx context.Context, url string) ([]Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Include a bit of the body: GitHub explains rate limits there.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("GET %s: %s: %s", url, resp.Status, body)
	}

	var events []Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("decode %s: %w", url, err)
	}
	return events, nil
}
