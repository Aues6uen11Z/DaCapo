package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/fynelabs/selfupdate"
)

// UpdateSource implements the selfupdate.Source interface
type UpdateSource struct {
	client         *http.Client
	versionURL     string
	appURL         string
	currentVersion string
}

// Ensure UpdateSource implements the Source interface
var _ selfupdate.Source = (*UpdateSource)(nil)

// NewUpdateSource creates a new custom Source
func NewUpdateSource(currentVersion string) *UpdateSource {
	client := &http.Client{Timeout: 5 * time.Minute}
	versionURL, appURL := getUrls()

	return &UpdateSource{
		client:         client,
		versionURL:     versionURL,
		appURL:         appURL,
		currentVersion: currentVersion,
	}
}

// Get implements the Source.Get method
// Downloads the updated executable file
func (c *UpdateSource) Get(v *selfupdate.Version) (io.ReadCloser, int64, error) {
	resp, err := c.client.Get(c.appURL)
	if err != nil {
		return nil, 0, err
	}

	return resp.Body, resp.ContentLength, nil
}

// GetSignature implements the Source.GetSignature method
// Returns the content of ${URL}.ed25519
func (c *UpdateSource) GetSignature() ([64]byte, error) {
	// Assume the signature file URL is the executable file URL with ".ed25519" appended
	signatureURL := c.appURL + ".ed25519"

	resp, err := c.client.Get(signatureURL)
	if err != nil {
		return [64]byte{}, fmt.Errorf("failed to get signature: %w", err)
	}
	defer resp.Body.Close()

	if resp.ContentLength != 64 {
		return [64]byte{}, fmt.Errorf("ed25519 signature must be 64 bytes long, got %d", resp.ContentLength)
	}

	signatureBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return [64]byte{}, fmt.Errorf("failed to read signature: %w", err)
	}

	if len(signatureBytes) != 64 {
		return [64]byte{}, fmt.Errorf("ed25519 signature must be 64 bytes long, got %d", len(signatureBytes))
	}

	var signature [64]byte
	copy(signature[:], signatureBytes)

	return signature, nil
}

// LatestVersion implements the Source.LatestVersion method
// Gets the latest version information to determine if an update should be triggered
func (c *UpdateSource) LatestVersion() (*selfupdate.Version, error) {
	// Get remote version number
	remoteVersion, err := c.getRemoteVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote version: %w", err)
	}

	// Parse current version
	currentVersion, err := semver.Parse(c.currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current version: %w", err)
	}

	// Create version information, using version string and constructing a pseudo timestamp
	// If remote version is newer, return a more recent timestamp
	var versionTime time.Time
	if remoteVersion.GT(currentVersion) {
		versionTime = time.Now().UTC()
	} else {
		// Return an older timestamp to indicate no update needed
		versionTime = time.Unix(0, 0).UTC()
	}

	return &selfupdate.Version{
		Number: remoteVersion.String(),
		Date:   versionTime,
	}, nil
}

// getRemoteVersion gets the remote latest version number
func (c *UpdateSource) getRemoteVersion() (semver.Version, error) {
	resp, err := c.client.Get(c.versionURL)
	if err != nil {
		return semver.Version{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return semver.Version{}, err
	}

	versionStr := strings.TrimSpace(string(body))
	return semver.Parse(versionStr)
}

// getCountry gets the current country location
func getCountry() (string, error) {
	resp, err := http.Get("http://ip-api.com/json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	country, ok := result["country"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get country from response")
	}

	return country, nil
}

// getUrls gets the URLs for version file and package, uses Gitee for China, GitHub for others
func getUrls() (string, string) {
	country, err := getCountry()
	if err != nil {
		log.Printf("Failed to get country: %v", err)
	}

	if country == "China" {
		return "https://gitee.com/aues6uen11z/da-capo/releases/download/latest/version.txt", "https://gitee.com/aues6uen11z/da-capo/releases/download/latest/DaCapo.exe"
	}
	return "https://github.com/Aues6uen11Z/DaCapo/releases/download/tools/version.txt", "https://github.com/Aues6uen11Z/DaCapo/releases/download/tools/DaCapo.exe"
}
