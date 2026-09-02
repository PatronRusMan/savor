package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	base   string
	client *http.Client
}

func New(base string) *Client {
	return &Client{base: base, client: &http.Client{Timeout: 5 * time.Second}}
}

type Restaurant struct {
	ID          string `json:"id"`
	OwnerID     string `json:"ownerId"`
	Name        string `json:"name"`
	IsOpen      bool   `json:"isOpen"`
	Address     string `json:"address"`
	ETAMinutes  int    `json:"etaMinutes"`
}

type Dish struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurantId"`
	Name         string `json:"name"`
	PriceCents   int    `json:"priceCents"`
	IsAvailable  bool   `json:"isAvailable"`
}

func (c *Client) Restaurant(id string) (Restaurant, error) {
	var out Restaurant
	err := c.get("/restaurants/"+id, &out)
	return out, err
}

func (c *Client) Dish(id string) (Dish, error) {
	var out Dish
	err := c.get("/dishes/"+id, &out)
	return out, err
}

func (c *Client) get(path string, dest any) error {
	resp, err := c.client.Get(c.base + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("catalog %s: status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}
