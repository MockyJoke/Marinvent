package dbf

import (
	"fmt"
	"os"
	"testing"
)

func TestAirportsDBFFields(t *testing.T) {
	airportsPath := "C:\\ProgramData\\Jeppesen\\Common\\TerminalCharts\\Airports.dbf"
	if _, err := os.Stat(airportsPath); os.IsNotExist(err) {
		t.Skip("Airports.dbf not found at", airportsPath)
	}

	db, err := New("", "", "", airportsPath)
	if err != nil {
		t.Fatalf("Failed to load airports.dbf: %v", err)
	}

	if db.airports == nil {
		t.Fatal("airports table is nil")
	}

	fields := db.airports.Fields()
	t.Logf("Number of fields: %d", len(fields))
	t.Log("Field names and indices:")
	for i, f := range fields {
		t.Logf("  [%d] %s (type=%s, len=%d)", i, f.Name, f.Type, f.Length)
	}
}

func TestAirportsDBFSampleData(t *testing.T) {
	airportsPath := "C:\\ProgramData\\Jeppesen\\Common\\TerminalCharts\\Airports.dbf"
	if _, err := os.Stat(airportsPath); os.IsNotExist(err) {
		t.Skip("Airports.dbf not found at", airportsPath)
	}

	db, err := New("", "", "", airportsPath)
	if err != nil {
		t.Fatalf("Failed to load airports.dbf: %v", err)
	}

	if db.airports == nil {
		t.Fatal("airports table is nil")
	}

	numRecords := db.airports.NumRecords()
	t.Logf("Number of records: %d", numRecords)

	fields := db.airports.Fields()

	t.Log("\n=== Sample data for first 5 records ===")
	for i := 0; i < 5 && i < numRecords; i++ {
		if db.airports.IsDeleted(i) {
			continue
		}
		t.Logf("\n--- Record %d ---", i)
		for _, f := range fields {
			val := db.airports.FieldValueByName(i, f.Name)
			if val != "" {
				t.Logf("  %s = %q", f.Name, val)
			}
		}
	}
}

func TestAirportsParsing(t *testing.T) {
	airportsPath := "C:\\ProgramData\\Jeppesen\\Common\\TerminalCharts\\Airports.dbf"
	if _, err := os.Stat(airportsPath); os.IsNotExist(err) {
		t.Skip("Airports.dbf not found at", airportsPath)
	}

	db, err := New("", "", "", airportsPath)
	if err != nil {
		t.Fatalf("Failed to load airports.dbf: %v", err)
	}

	testICAOs := []string{"KJFK", "KLAX", "EGLL", "LFPG", "OMDB"}
	for _, icao := range testICAOs {
		airport := db.GetAirport(icao)
		if airport == nil {
			t.Logf("Airport %s: not found", icao)
			continue
		}
		t.Logf("\n=== %s ===", icao)
		t.Logf("  ICAO: %s", airport.ICAO)
		t.Logf("  IATA: %s", airport.IATA)
		t.Logf("  Name: %s", airport.Name)
		t.Logf("  City: %s", airport.City)
		t.Logf("  State: %s", airport.State)
		t.Logf("  Country: %s", airport.CountryCode)
		t.Logf("  Lat/Lon: %.6f, %.6f", airport.Latitude, airport.Longitude)
		t.Logf("  Mag Var: %.2f", airport.MagneticVariation)
		t.Logf("  Longest Runway: %d ft", airport.LongestRunwayFt)
		t.Logf("  Timezone: %s", airport.Timezone)
		t.Logf("  Airport Use: %s", airport.AirportUse)
		t.Logf("  Customs: %s", airport.Customs)
		t.Logf("  Beacon: %v", airport.Beacon)
		t.Logf("  Jet Start Unit: %v", airport.JetStartUnit)
		t.Logf("  Oxygen: %v", airport.Oxygen)
		t.Logf("  Repair Types: %v", airport.RepairTypes)
		t.Logf("  Fuel Types: %v", airport.FuelTypes)
	}
}

func TestDumpAirportsFieldsForDoc(t *testing.T) {
	airportsPath := "C:\\ProgramData\\Jeppesen\\Common\\TerminalCharts\\Airports.dbf"
	if _, err := os.Stat(airportsPath); os.IsNotExist(err) {
		t.Skip("Airports.dbf not found at", airportsPath)
	}

	db, err := New("", "", "", airportsPath)
	if err != nil {
		t.Fatalf("Failed to load airports.dbf: %v", err)
	}

	fields := db.airports.Fields()
	fmt.Println("\n=== Field list for documentation ===")
	for i, f := range fields {
		fmt.Printf("[%d] %s\n", i, f.Name)
	}

	fmt.Println("\n=== Sample record with all fields ===")
	for i := 0; i < db.airports.NumRecords(); i++ {
		if db.airports.IsDeleted(i) {
			continue
		}
		fmt.Printf("\nRecord %d:\n", i)
		for j, f := range fields {
			val := db.airports.FieldValueByName(i, f.Name)
			fmt.Printf("  [%d] %s = %q\n", j, f.Name, val)
		}
		break
	}
}
