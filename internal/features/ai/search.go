package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type SearchClient struct {
	endpoint string
	key      string
	hc       *http.Client
}

func NewSearchClient(endpoint, key string, hc *http.Client) *SearchClient {
	return &SearchClient{endpoint: endpoint, key: key, hc: hc}
}

func (c *SearchClient) Search(ctx context.Context, query string) ([]SearchResult, error) {
	u := c.endpoint + "?" + url.Values{"q": {query}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search engine mengembalikan status %d", resp.StatusCode)
	}
	var body struct {
		Results []SearchResult `json:"results"`
		Data    []SearchResult `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if len(body.Results) == 0 {
		return body.Data, nil
	}
	return body.Results, nil
}
