package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"marinvent/internal/charts"
	"marinvent/internal/georef"

	"github.com/gin-gonic/gin"
)

// @title Marinvent Chart API
// @version 1.0
// @description API for accessing and exporting Jeppesen terminal charts
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/marinvent/marivent
// @contact.email support@marinvent.local

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// ChartInfo is the API response model
type ChartInfo struct {
	Filename  string `json:"filename"`
	ICAO      string `json:"icao"`
	ChartType string `json:"chart_type"`
	TypeName  string `json:"type_name"`
	Category  string `json:"category"`
	ProcID    string `json:"proc_id"`
	DateEff   string `json:"date_eff"`
	SheetID   string `json:"sheet_id"`
	HasTCL    bool   `json:"has_tcl"`
	IsVFR     bool   `json:"is_vfr"`
}

// ChartList is the API response for listing charts
// @description Response containing list of charts for an ICAO
type ChartList struct {
	ICAO   string      `json:"icao"`
	Total  int         `json:"total"`
	Charts []ChartInfo `json:"charts"`
}

// ChartTypesResponse is the API response for listing chart types
type ChartTypesResponse struct {
	Total int         `json:"total"`
	Types []ChartType `json:"types"`
}

// ChartType is a chart type entry
type ChartType struct {
	Code     string `json:"code"`
	Category string `json:"category"`
	Type     string `json:"type"`
}

// Airport is the API response model for airport info
type Airport struct {
	ICAO              string   `json:"icao"`
	IATA              string   `json:"iata,omitempty"`
	Name              string   `json:"name"`
	City              string   `json:"city,omitempty"`
	State             string   `json:"state,omitempty"`
	CountryCode       string   `json:"country_code,omitempty"`
	Latitude          float64  `json:"latitude"`
	Longitude         float64  `json:"longitude"`
	MagneticVariation float64  `json:"magnetic_variation"`
	LongestRunwayFt   int      `json:"longest_runway_ft"`
	Timezone          string   `json:"timezone,omitempty"`
	AirportUse        string   `json:"airport_use"`
	Customs           string   `json:"customs,omitempty"`
	Beacon            bool     `json:"beacon"`
	JetStartUnit      bool     `json:"jet_start_unit"`
	Oxygen            []string `json:"oxygen,omitempty"`
	RepairTypes       []string `json:"repair_types,omitempty"`
	FuelTypes         []string `json:"fuel_types,omitempty"`
}

// Config holds API configuration
type Config struct {
	ChartsDBFPath string
	TypesDBFPath  string
	TCLDir        string
}

// Handler holds the catalog and config
type Handler struct {
	catalog   *charts.Catalog
	config    *Config
	geoClient *georef.Client
}

// NewHandler creates a new API handler
func NewHandler(catalog *charts.Catalog, config *Config) *Handler {
	return &Handler{
		catalog:   catalog,
		config:    config,
		geoClient: georef.NewClient(config.TCLDir),
	}
}

// GetCharts returns charts for an ICAO
// @Summary List charts for ICAO
// @Description Returns all charts for a given ICAO airport. Can be filtered by type (code or name) and search query.
// @Tags charts
// @Accept json
// @Produce json
// @Param icao path string true "ICAO airport code (e.g., KJFK, EGLL)"
// @Param type query string false "Chart type - can be code (1L, AP) or name (RNAV, ILS, AIRPORT). Looks up in ctypes.dbf"
// @Param search query string false "Search text to filter by PROC_ID (procedure name)"
// @Param types query string false "Chart types to include - can be 'vfr', 'ifr', or 'vfr,ifr' (default: both)"
// @Success 200 {object} ChartList
// @Router /api/v1/charts/{icao} [get]
func (h *Handler) GetCharts(c *gin.Context) {
	start := time.Now()

	icao := c.Param("icao")
	typeQuery := c.Query("type")
	search := c.Query("search")
	types := c.Query("types")

	t := time.Now()
	results := h.catalog.Filter(icao, typeQuery, search, types)
	filterTime := time.Since(t)

	chartList := make([]ChartInfo, 0, len(results))
	for _, r := range results {
		chartList = append(chartList, ChartInfo{
			Filename:  r.Filename,
			ICAO:      r.ICAO,
			ChartType: r.ChartType,
			TypeName:  r.TypeName,
			Category:  r.Category,
			ProcID:    r.ProcID,
			DateEff:   r.DateEff,
			SheetID:   r.SheetID,
			HasTCL:    r.TCLPath != "",
			IsVFR:     r.IsVFR,
		})
	}

	c.JSON(http.StatusOK, ChartList{
		ICAO:   icao,
		Total:  len(chartList),
		Charts: chartList,
	})

	log.Printf("[PERF] GetCharts(%s): filter=%.1fms total=%.1fms results=%d",
		icao, float64(filterTime.Microseconds())/1000, float64(time.Since(start).Microseconds())/1000, len(chartList))
}

// GetChartTypes returns all available chart types
// @Summary List all chart types
// @Description Returns all available chart types from ctypes.dbf with codes, categories, and descriptions
// @Tags charts
// @Accept json
// @Produce json
// @Success 200 {object} ChartTypesResponse
// @Router /api/v1/chart-types [get]
func (h *Handler) GetChartTypes(c *gin.Context) {
	db := h.catalog.GetDBF()
	types := db.GetAllChartTypes()

	typeList := make([]ChartType, 0, len(types))
	for _, t := range types {
		typeList = append(typeList, ChartType{
			Code:     t.Code,
			Category: t.Category,
			Type:     t.Type,
		})
	}

	c.JSON(http.StatusOK, ChartTypesResponse{
		Total: len(typeList),
		Types: typeList,
	})
}

// GetAirport gets an airport by ICAO code
// @Summary Get airport by ICAO
// @Description Returns airport details for a given ICAO code
// @Tags airports
// @Accept json
// @Produce json
// @Param icao path string true "ICAO airport code (e.g., KJFK, EGLL)"
// @Success 200 {object} Airport
// @Router /api/v1/airports/{icao} [get]
func (h *Handler) GetAirport(c *gin.Context) {
	icao := c.Param("icao")
	icao = strings.ToUpper(icao)

	airport := h.catalog.GetAirport(icao)
	if airport == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "airport not found"})
		return
	}

	c.JSON(http.StatusOK, Airport{
		ICAO:              airport.ICAO,
		IATA:              airport.IATA,
		Name:              airport.Name,
		City:              airport.City,
		State:             airport.State,
		CountryCode:       airport.CountryCode,
		Latitude:          airport.Latitude,
		Longitude:         airport.Longitude,
		MagneticVariation: airport.MagneticVariation,
		LongestRunwayFt:   airport.LongestRunwayFt,
		Timezone:          airport.Timezone,
		AirportUse:        airport.AirportUse,
		Customs:           airport.Customs,
		Beacon:            airport.Beacon,
		JetStartUnit:      airport.JetStartUnit,
		Oxygen:            airport.Oxygen,
		RepairTypes:       airport.RepairTypes,
		FuelTypes:         airport.FuelTypes,
	})
}

// GetChartPDF exports a chart to PDF
// @Summary Export chart to PDF
// @Description Exports a specific chart to PDF format. Returns the chart as a PDF file. Post-processing is enabled by default to remove waypoint overlays; use ?no_postprocess=1 to disable.
// @Tags charts
// @Accept json
// @Produce application/pdf
// @Param icao path string true "ICAO airport code"
// @Param filename path string true "Chart filename (e.g., KJFK225)"
// @Param no_postprocess query int false "Set to 1 to disable post-processing (default: 0)"
// @Success 200 {file} pdf "PDF file containing the chart"
// @Router /api/v1/charts/{icao}/export/{filename} [get]
func (h *Handler) GetChartPDF(c *gin.Context) {
	start := time.Now()
	timings := make(map[string]time.Duration)
	tick := func(name string, t time.Time) time.Duration {
		d := time.Since(t)
		timings[name] = d
		return d
	}

	t := time.Now()
	icao := c.Param("icao")
	filename := c.Param("filename")
	noPostProcess := c.Query("no_postprocess") == "1"
	tick("params", t)

	t = time.Now()
	chart := h.catalog.GetChart(filename)
	tick("GetChart", t)

	if chart == nil || chart.ICAO != icao {
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
		return
	}

	if chart.TCLPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "TCL file not found"})
		return
	}

	t = time.Now()
	pdfBytes, err := h.catalog.ExportToPDF(chart.TCLPath, !noPostProcess)
	tick("ExportToPDF", t)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	t = time.Now()
	c.Header("Content-Disposition", "attachment; filename="+filename+".pdf")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
	tick("response", t)

	timings["total"] = time.Since(start)

	logTimings(timings, filename)
}

func logTimings(timings map[string]time.Duration, filename string) {
	total := timings["total"]
	delete(timings, "total")

	parts := make([]string, 0, len(timings)+1)
	for name, d := range timings {
		parts = append(parts, fmt.Sprintf("%s=%.1fms", name, float64(d.Microseconds())/1000))
	}
	parts = append(parts, fmt.Sprintf("total=%.1fms", float64(total.Microseconds())/1000))

	log.Printf("[PERF] %s: %s", filename, strings.Join(parts, " "))
}

// GetHealth returns health check
// ChartDataResponse is the API response for getting chart data (api.ChartDataResponse)
type ChartDataResponse struct {
	Filename string                `json:"filename"`
	ICAO     string                `json:"icao"`
	Width    int32                 `json:"width"`
	Height   int32                 `json:"height"`
	HasTCL   bool                  `json:"has_tcl"`
	Georef   *GeoRefStatusResponse `json:"georef,omitempty"`
}

// GetChartData returns data for a single chart
// @Summary Get chart data
// @Description Returns chart data including dimensions and georeferencing status for a specific chart
// @Tags charts
// @Accept json
// @Produce json
// @Param icao path string true "ICAO airport code"
// @Param filename path string true "Chart filename (e.g., KJFK225)"
// @Success 200 {object} ChartDataResponse
// @Router /api/v1/charts/{icao}/data/{filename} [get]
func (h *Handler) GetChartData(c *gin.Context) {
	icao := c.Param("icao")
	filename := c.Param("filename")

	chart := h.catalog.GetChart(filename)
	if chart == nil || chart.ICAO != icao {
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
		return
	}

	response := ChartDataResponse{
		Filename: chart.Filename,
		ICAO:     chart.ICAO,
		Width:    0,
		Height:   0,
		HasTCL:   chart.TCLPath != "",
	}

	if chart.TCLPath != "" {
		status, err := h.geoClient.GetStatus(chart.TCLPath)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(status)
		}
		if err == nil && status != nil {
			if status.Georeferenced {
				response.Georef = &GeoRefStatusResponse{
					Georeferenced: true,
					Bounds: &ChartBoundsResponse{
						Left:   status.Bounds.Left,
						Top:    status.Bounds.Top,
						Right:  status.Bounds.Right,
						Bottom: status.Bounds.Bottom,
						Width:  status.Bounds.Width,
						Height: status.Bounds.Height,
					},
				}
				response.Width = status.Bounds.Width
				response.Height = status.Bounds.Height
			} else {
				response.Georef = &GeoRefStatusResponse{
					Georeferenced: false,
				}
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Health check
// @Description Returns API health status and version
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *Handler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": "2.3.0",
	})
}

// CoordToPixelRequest is the request for coordinate to pixel conversion
type CoordToPixelRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// CoordToPixelResponse is the response for coordinate to pixel conversion
type CoordToPixelResponse struct {
	X     int    `json:"x,omitempty"`
	Y     int    `json:"y,omitempty"`
	Error string `json:"error,omitempty"`
}

// PixelToCoordRequest is the request for pixel to coordinate conversion
type PixelToCoordRequest struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// PixelToCoordResponse is the response for pixel to coordinate conversion
type PixelToCoordResponse struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// BatchCoordToPixelRequest is a batch request for coordinate conversions
type BatchCoordToPixelRequest struct {
	Points []CoordToPixelRequest `json:"points"`
}

// BatchCoordToPixelResponse is a batch response for coordinate conversions
type BatchCoordToPixelResponse struct {
	Points []CoordToPixelResponse `json:"points"`
}

// BatchPixelToCoordRequest is a batch request for pixel conversions
type BatchPixelToCoordRequest struct {
	Points []PixelToCoordRequest `json:"points"`
}

// BatchPixelToCoordResponse is a batch response for pixel conversions
type BatchPixelToCoordResponse struct {
	Points []PixelToCoordResponse `json:"points"`
}

// ChartBoundsResponse is the response for chart bounds
type ChartBoundsResponse struct {
	Left   int32 `json:"left"`
	Top    int32 `json:"top"`
	Right  int32 `json:"right"`
	Bottom int32 `json:"bottom"`
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

// GeoRefStatusResponse is the response for georeferencing status
type GeoRefStatusResponse struct {
	Georeferenced bool                 `json:"georeferenced"`
	Bounds        *ChartBoundsResponse `json:"bounds,omitempty"`
}

// GetGeoRefStatus returns georeferencing status for a chart
// @Summary Get chart georeferencing status
// @Description Returns whether a chart is georeferenced and its pixel bounds
// @Tags georef
// @Accept json
// @Produce json
// @Param icao path string true "ICAO airport code"
// @Param filename path string true "Chart filename (e.g., KJFK225)"
// @Success 200 {object} GeoRefStatusResponse
// @Router /api/v1/charts/{icao}/geo/status/{filename} [get]
func (h *Handler) GetGeoRefStatus(c *gin.Context) {
	icao := c.Param("icao")
	filename := c.Param("filename")

	chart := h.catalog.GetChart(filename)
	if chart == nil || chart.ICAO != icao {
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
		return
	}

	if chart.TCLPath == "" {
		c.JSON(http.StatusOK, GeoRefStatusResponse{
			Georeferenced: false,
		})
		return
	}

	status, err := h.geoClient.GetStatus(chart.TCLPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var bounds *ChartBoundsResponse
	if status.Georeferenced {
		bounds = &ChartBoundsResponse{
			Left:   status.Bounds.Left,
			Top:    status.Bounds.Top,
			Right:  status.Bounds.Right,
			Bottom: status.Bounds.Bottom,
			Width:  status.Bounds.Width,
			Height: status.Bounds.Height,
		}
	}

	c.JSON(http.StatusOK, GeoRefStatusResponse{
		Georeferenced: status.Georeferenced,
		Bounds:        bounds,
	})
}

// CoordToPixel converts geographic coordinates to pixel coordinates
// @Summary Convert coordinates to pixel
// @Description Converts geographic coordinates (lat, lon) to chart pixel coordinates (x, y)
// @Tags georef
// @Accept json
// @Produce json
// @Param icao path string true "ICAO airport code"
// @Param filename path string true "Chart filename"
// @Param request body CoordToPixelRequest true "Geographic coordinates"
// @Success 200 {object} CoordToPixelResponse
// @Router /api/v1/charts/{icao}/geo/coord2pixel/{filename} [post]
func (h *Handler) CoordToPixel(c *gin.Context) {
	icao := c.Param("icao")
	filename := c.Param("filename")

	chart := h.catalog.GetChart(filename)
	if chart == nil || chart.ICAO != icao {
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
		return
	}

	if chart.TCLPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "TCL file not found"})
		return
	}

	var req CoordToPixelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.geoClient.CoordToPixel(chart.TCLPath, req.Latitude, req.Longitude)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, CoordToPixelResponse{
		X:     result.X,
		Y:     result.Y,
		Error: result.Error,
	})
}

// PixelToCoord converts pixel coordinates to geographic coordinates
// @Summary Convert pixel to coordinates
// @Description Converts chart pixel coordinates (x, y) to geographic coordinates (lat, lon)
// @Tags georef
// @Accept json
// @Produce json
// @Param icao path string true "ICAO airport code"
// @Param filename path string true "Chart filename"
// @Param request body PixelToCoordRequest true "Pixel coordinates"
// @Success 200 {object} PixelToCoordResponse
// @Router /api/v1/charts/{icao}/geo/pixel2coord/{filename} [post]
func (h *Handler) PixelToCoord(c *gin.Context) {
	icao := c.Param("icao")
	filename := c.Param("filename")

	chart := h.catalog.GetChart(filename)
	if chart == nil || chart.ICAO != icao {
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
		return
	}

	if chart.TCLPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "TCL file not found"})
		return
	}

	var req PixelToCoordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.geoClient.PixelToCoord(chart.TCLPath, req.X, req.Y)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, PixelToCoordResponse{
		Latitude:  result.Latitude,
		Longitude: result.Longitude,
		Error:     result.Error,
	})
}

// BatchCoordToPixel batch converts coordinates to pixels
// @Summary Batch convert coordinates to pixels
// @Description Converts multiple geographic coordinates to pixel coordinates
// @Tags georef
// @Accept json
// @Produce json
// @Param icao path string true "ICAO airport code"
// @Param filename path string true "Chart filename"
// @Param request body BatchCoordToPixelRequest true "Geographic coordinates"
// @Success 200 {object} BatchCoordToPixelResponse
// @Router /api/v1/charts/{icao}/geo/batch-coord2pixel/{filename} [post]
func (h *Handler) BatchCoordToPixel(c *gin.Context) {
	icao := c.Param("icao")
	filename := c.Param("filename")

	chart := h.catalog.GetChart(filename)
	if chart == nil || chart.ICAO != icao {
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
		return
	}

	if chart.TCLPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "TCL file not found"})
		return
	}

	var req BatchCoordToPixelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	coords := make([]georef.CoordRequest, len(req.Points))
	for i, p := range req.Points {
		coords[i] = georef.CoordRequest{Latitude: p.Latitude, Longitude: p.Longitude}
	}

	results, err := h.geoClient.BatchCoordToPixel(chart.TCLPath, coords)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	points := make([]CoordToPixelResponse, len(results))
	for i, r := range results {
		points[i] = CoordToPixelResponse{X: r.X, Y: r.Y, Error: r.Error}
	}

	c.JSON(http.StatusOK, BatchCoordToPixelResponse{Points: points})
}

// BatchPixelToCoord batch converts pixels to coordinates
// @Summary Batch convert pixels to coordinates
// @Description Converts multiple pixel coordinates to geographic coordinates
// @Tags georef
// @Accept json
// @Produce json
// @Param icao path string true "ICAO airport code"
// @Param filename path string true "Chart filename"
// @Param request body BatchPixelToCoordRequest true "Pixel coordinates"
// @Success 200 {object} BatchPixelToCoordResponse
// @Router /api/v1/charts/{icao}/geo/batch-pixel2coord/{filename} [post]
func (h *Handler) BatchPixelToCoord(c *gin.Context) {
	icao := c.Param("icao")
	filename := c.Param("filename")

	chart := h.catalog.GetChart(filename)
	if chart == nil || chart.ICAO != icao {
		c.JSON(http.StatusNotFound, gin.H{"error": "chart not found"})
		return
	}

	if chart.TCLPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "TCL file not found"})
		return
	}

	var req BatchPixelToCoordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	pixels := make([]georef.PixelRequest, len(req.Points))
	for i, p := range req.Points {
		pixels[i] = georef.PixelRequest{X: p.X, Y: p.Y}
	}

	results, err := h.geoClient.BatchPixelToCoord(chart.TCLPath, pixels)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	points := make([]PixelToCoordResponse, len(results))
	for i, r := range results {
		points[i] = PixelToCoordResponse{Latitude: r.Latitude, Longitude: r.Longitude, Error: r.Error}
	}

	c.JSON(http.StatusOK, BatchPixelToCoordResponse{Points: points})
}
