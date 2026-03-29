package network

import "github.com/jonshaffer/go-unifi/unifi"

// App provides methods for the UniFi Network application.
// All Network-specific SDK methods are attached to this type.
type App struct {
	Client *unifi.Client
	Site   string // site name (e.g., "default")
}

// NewApp creates a Network application client.
func NewApp(client *unifi.Client, site string) *App {
	return &App{Client: client, Site: site}
}
