package digitalocean

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// Client wraps the DigitalOcean API client
type Client struct {
	client *godo.Client
	logger *zap.Logger
}

// TokenSource implements oauth2.TokenSource for DigitalOcean API authentication
type TokenSource struct {
	AccessToken string
}

// Token returns the oauth2 token for API authentication
func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

// NewClient creates a new DigitalOcean client
func NewClient(apiKey string, logger *zap.Logger) *Client {
	tokenSource := &TokenSource{
		AccessToken: apiKey,
	}

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := godo.NewClient(oauthClient)

	return &Client{
		client: client,
		logger: logger.Named("digitalocean"),
	}
}

// GetFirewall retrieves a firewall by ID
func (c *Client) GetFirewall(ctx context.Context, firewallID string) (*godo.Firewall, error) {
	c.logger.Debug("Getting firewall", zap.String("firewall_id", firewallID))

	firewall, _, err := c.client.Firewalls.Get(ctx, firewallID)
	if err != nil {
		c.logger.Error("Failed to get firewall",
			zap.String("firewall_id", firewallID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get firewall %s: %w", firewallID, err)
	}

	c.logger.Debug("Successfully retrieved firewall",
		zap.String("firewall_id", firewallID),
		zap.String("firewall_name", firewall.Name))

	return firewall, nil
}
