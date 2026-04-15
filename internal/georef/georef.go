package georef

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"marinvent/internal/runtimepaths"
)

type GeoRefStatus struct {
	Georeferenced bool        `json:"georeferenced"`
	Bounds        ChartBounds `json:"bounds"`
	Error         string      `json:"error,omitempty"`
}

type ChartBounds struct {
	Left   int32 `json:"left"`
	Top    int32 `json:"top"`
	Right  int32 `json:"right"`
	Bottom int32 `json:"bottom"`
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

type PixelCoord struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Error string `json:"error,omitempty"`
}

type GeoCoord struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Error     string  `json:"error,omitempty"`
}

type Client struct {
	toolPath string
	tclDir   string
	mu       sync.Mutex
}

func NewClient(tclDir string) *Client {
	toolPath := runtimepaths.DefaultToolPath("georef_tool.exe")
	if !filepath.IsAbs(toolPath) {
		absPath, err := filepath.Abs(toolPath)
		if err == nil {
			toolPath = absPath
		}
	}

	return &Client{
		toolPath: toolPath,
		tclDir:   tclDir,
	}
}

func (c *Client) getToolPath() string {
	return c.toolPath
}

func (c *Client) runTool(args ...string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cmd, err := runtimepaths.PrepareCommand(c.getToolPath(), args...)
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tool failed: %w, output: %s", err, string(output))
	}

	return output, nil
}

func (c *Client) GetStatus(tclPath string) (*GeoRefStatus, error) {
	runtimeTCLPath, err := runtimepaths.RuntimeFilePath(tclPath)
	if err != nil {
		return nil, err
	}

	output, err := c.runTool("status", runtimeTCLPath)
	if err != nil {
		return nil, err
	}

	var status GeoRefStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &status, nil
}

func (c *Client) CoordToPixel(tclPath string, lat, lon float64) (*PixelCoord, error) {
	runtimeTCLPath, err := runtimepaths.RuntimeFilePath(tclPath)
	if err != nil {
		return nil, err
	}

	args := []string{
		"coord2pixel",
		runtimeTCLPath,
		fmt.Sprintf("%.10f", lat),
		fmt.Sprintf("%.10f", lon),
	}

	output, err := c.runTool(args...)
	if err != nil {
		return nil, err
	}

	var result PixelCoord
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (c *Client) PixelToCoord(tclPath string, x, y int) (*GeoCoord, error) {
	runtimeTCLPath, err := runtimepaths.RuntimeFilePath(tclPath)
	if err != nil {
		return nil, err
	}

	args := []string{
		"pixel2coord",
		runtimeTCLPath,
		fmt.Sprintf("%d", x),
		fmt.Sprintf("%d", y),
	}

	output, err := c.runTool(args...)
	if err != nil {
		return nil, err
	}

	var result GeoCoord
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (c *Client) BatchCoordToPixel(tclPath string, coords []CoordRequest) ([]PixelCoord, error) {
	results := make([]PixelCoord, len(coords))
	for i, coord := range coords {
		result, err := c.CoordToPixel(tclPath, coord.Latitude, coord.Longitude)
		if err != nil {
			results[i] = PixelCoord{Error: err.Error()}
		} else {
			results[i] = *result
		}
	}
	return results, nil
}

func (c *Client) BatchPixelToCoord(tclPath string, pixels []PixelRequest) ([]GeoCoord, error) {
	results := make([]GeoCoord, len(pixels))
	for i, pixel := range pixels {
		result, err := c.PixelToCoord(tclPath, pixel.X, pixel.Y)
		if err != nil {
			results[i] = GeoCoord{Error: err.Error()}
		} else {
			results[i] = *result
		}
	}
	return results, nil
}

type CoordRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type PixelRequest struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func (c *Client) ResolveTCLPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}

	if c.tclDir != "" {
		fullPath := filepath.Join(c.tclDir, filename)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}

		if !strings.HasSuffix(strings.ToLower(filename), ".tcl") {
			fullPath = filepath.Join(c.tclDir, filename+".tcl")
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath
			}
		}
	}

	return filename
}
