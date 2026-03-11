package main

import (
	"flag"
	"fmt"
	"os"

	"marinvent/internal/api"
)

func main() {
	port := flag.String("port", "8080", "API server port")
	host := flag.String("host", "0.0.0.0", "API server host")
	chartsDBF := flag.String("charts", "C:\\ProgramData\\Jeppesen\\Common\\TerminalCharts\\charts.dbf", "Path to charts.dbf")
	vfrChartsDBF := flag.String("vfrcharts", "C:\\ProgramData\\Jeppesen\\Common\\TerminalCharts\\vfrchrts.dbf", "Path to vfrchrts.dbf")
	typesDBF := flag.String("types", "C:\\ProgramData\\Jeppesen\\Common\\TerminalCharts\\ctypes.dbf", "Path to ctypes.dbf")
	airportsDBF := flag.String("airports", "C:\\ProgramData\\Jeppesen\\Common\\TerminalCharts\\Airports.dbf", "Path to Airports.dbf")
	tclDir := flag.String("tcls", "TCLs", "Directory containing TCL files")
	flag.Parse()

	cfg := api.ServerConfig{
		Host:         *host,
		Port:         *port,
		ChartsDBF:    *chartsDBF,
		VFRChartsDBF: *vfrChartsDBF,
		TypesDBF:     *typesDBF,
		AirportsDBF:  *airportsDBF,
		TCLDir:       *tclDir,
	}

	server, err := api.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
