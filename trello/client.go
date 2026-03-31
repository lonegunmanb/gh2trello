package trello

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/v69/github"
)

// Client wraps the Trello API client.
type Client struct {
	APIKey   string
	Token    string
	BaseURL  string
	httpClient *http.Client
}

// Card represents a Trello card.
type Card struct {
	ID       string `json:"id"`
	Name     string  `json:"name"`
	Desc     string  `json:"desc"`
	IDList   string  `json:"idList"`
	Pos      float64 `json:"pos,omitempty"`
}

// NewClient creates a new Trello client.
func NewClient(apiKey, token string) *Client {
	return &Client{
		APIKey:   apiKey,
		Token:    token,
		BaseURL:  "https://api.trello.com/1",
		httpClient: http.DefaultClient,
	}
}

// SetHTTPClient sets a custom HTTP client (useful for testing).
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// CreateCard creates a new card in the specified list.
func (c *Client) CreateCard(listID, name, desc string) (*Card, error) {
	url := fmt.Sprintf("%s/cards?key=%s&token=%s", c.BaseURL, c.APIKey, c.Token)

	data := map[string]string{
		"idList": listID,
		"name":   name,
		"desc":   desc,
		"pos":    "bottom",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create card: %s", string(body))
	}

	var card Card
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, err
	}

	return &card, nil
}

// GetCardsInList retrieves all cards in a list.
func (c *Client) GetCardsInList(listID string) ([]Card, error) {
	url := fmt.Sprintf("%s/lists/%s/cards?key=%s&token=%s&fields=desc", c.BaseURL, listID, c.APIKey, c.Token)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get cards: %s", string(body))
	}

	var cards []Card
	if err := json.NewDecoder(resp.Body).Decode(&cards); err != nil {
		return nil, err
	}

	return cards, nil
}

// CardExists checks if a card with the given URL already exists in the list.
func (c *Client) CardExists(listID, url string) (bool, error) {
	cards, err := c.GetCardsInList(listID)
	if err != nil {
		return false, err
	}

	for _, card := range cards {
		if strings.HasPrefix(card.Desc, url) {
			return true, nil
		}
	}

	return false, nil
}

// FindCard searches for a card with the given URL in either the issue list or
// the PR list. It returns the card with its IDList field set to the list where
// it was found, or nil if the card does not exist in either list.
func (c *Client) FindCard(issueListID, prListID, url string) (*Card, error) {
	for _, listID := range []string{issueListID, prListID} {
		cards, err := c.GetCardsInList(listID)
		if err != nil {
			return nil, err
		}
		for _, card := range cards {
			if strings.HasPrefix(card.Desc, url) {
				card.IDList = listID
				return &card, nil
			}
		}
	}
	return nil, nil
}

// MoveCard moves a card to a different list.
func (c *Client) MoveCard(cardID, targetListID string) error {
	url := fmt.Sprintf("%s/cards/%s?key=%s&token=%s", c.BaseURL, cardID, c.APIKey, c.Token)

	data := map[string]string{
		"idList": targetListID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to move card: %s", string(body))
	}

	return nil
}

// CreateCardForIssue creates a Trello card for a GitHub issue or PR,
// dispatching to the correct list based on whether it's a PR.
func (c *Client) CreateCardForIssue(issue *github.Issue, issueListID, prListID string) (*Card, error) {
	name := FormatCardName(issue)
	desc := FormatCardDesc(issue)

	listID := issueListID
	if issue.PullRequestLinks != nil {
		listID = prListID
	}

	return c.CreateCard(listID, name, desc)
}