package dbf

import (
	"fmt"

	"github.com/Bowbaq/dbf"
)

type Chart struct {
	ICAO      string
	Filename  string
	ChartType string
	IndexNo   string
	ProcID    string
	Action    string
	DateRev   string
	DateEff   string
	TrimSize  string
	GeoRef    string
	SheetID   string
	FtBk      string
}

type ChartType struct {
	Code      string
	Category  string
	Type      string
	Precision string
}

type Airport struct {
	ICAO              string
	IATA              string
	Name              string
	City              string
	State             string
	CountryCode       string
	Latitude          float64
	Longitude         float64
	MagneticVariation float64
	LongestRunwayFt   int
	Timezone          string
	AirportUse        string
	Customs           string
	Beacon            bool
	JetStartUnit      bool
	Oxygen            []string
	RepairTypes       []string
	FuelTypes         []string
}

type DBF struct {
	charts     *dbf.DbfTable
	ctypes     *dbf.DbfTable
	airports   *dbf.DbfTable
	chartMap   map[string]*Chart
	typeMap    map[string]*ChartType
	airportMap map[string]*Airport
}

func New(chartsPath, ctypesPath, airportsPath string) (*DBF, error) {
	var charts, ctypes *dbf.DbfTable
	var err error

	if chartsPath != "" {
		charts, err = dbf.LoadFile(chartsPath)
		if err != nil {
			return nil, err
		}
	}

	if ctypesPath != "" {
		ctypes, err = dbf.LoadFile(ctypesPath)
		if err != nil {
			return nil, err
		}
	}

	var airports *dbf.DbfTable
	if airportsPath != "" {
		airports, err = dbf.LoadFile(airportsPath)
		if err != nil {
			return nil, err
		}
	}

	d := &DBF{
		charts:     charts,
		ctypes:     ctypes,
		airports:   airports,
		chartMap:   make(map[string]*Chart),
		typeMap:    make(map[string]*ChartType),
		airportMap: make(map[string]*Airport),
	}

	d.buildMaps()
	return d, nil
}

func (d *DBF) buildMaps() {
	if d.charts != nil {
		iter := d.charts.NewIterator()
		for iter.Next() {
			row := iter.Row()
			if len(row) >= 12 {
				chart := &Chart{
					ICAO:      trim(row[0]),
					Filename:  trim(row[1]),
					ChartType: trim(row[2]),
					IndexNo:   trim(row[3]),
					ProcID:    trim(row[4]),
					Action:    trim(row[5]),
					DateRev:   trim(row[6]),
					DateEff:   trim(row[7]),
					TrimSize:  trim(row[8]),
					GeoRef:    trim(row[9]),
					SheetID:   trim(row[10]),
					FtBk:      trim(row[11]),
				}
				d.chartMap[chart.Filename] = chart
			}
		}
	}

	if d.ctypes != nil {
		iter := d.ctypes.NewIterator()
		for iter.Next() {
			row := iter.Row()
			if len(row) >= 4 {
				ct := &ChartType{
					Code:      trim(row[0]),
					Category:  trim(row[1]),
					Type:      trim(row[2]),
					Precision: trim(row[3]),
				}
				d.typeMap[ct.Code] = ct
			}
		}
	}

	if d.airports != nil {
		for i := 0; i < d.airports.NumRecords(); i++ {
			if d.airports.IsDeleted(i) {
				continue
			}
			airport := d.parseAirportByRow(i)
			if airport != nil && airport.ICAO != "" {
				d.airportMap[airport.ICAO] = airport
			}
		}
	}
}

func (d *DBF) GetChart(filename string) *Chart {
	return d.chartMap[filename]
}

func (d *DBF) GetChartType(code string) *ChartType {
	return d.typeMap[code]
}

func (d *DBF) GetAllCharts() []*Chart {
	charts := make([]*Chart, 0, len(d.chartMap))
	for _, c := range d.chartMap {
		charts = append(charts, c)
	}
	return charts
}

func (d *DBF) GetAllChartTypes() []*ChartType {
	types := make([]*ChartType, 0, len(d.typeMap))
	for _, t := range d.typeMap {
		types = append(types, t)
	}
	return types
}

func (d *DBF) SearchCharts(query string) []*Chart {
	query = toUpper(query)
	var results []*Chart
	for _, c := range d.chartMap {
		if contains(c.ICAO, query) || contains(c.Filename, query) || contains(c.ProcID, query) {
			results = append(results, c)
		}
	}
	return results
}

func (d *DBF) FilterByType(chartType string) []*Chart {
	var results []*Chart
	for _, c := range d.chartMap {
		if c.ChartType == chartType {
			results = append(results, c)
		}
	}
	return results
}

func (d *DBF) FilterByICAO(icao string) []*Chart {
	icao = toUpper(icao)
	var results []*Chart
	for _, c := range d.chartMap {
		if c.ICAO == icao {
			results = append(results, c)
		}
	}
	return results
}

func (d *DBF) NumCharts() int {
	return len(d.chartMap)
}

func (d *DBF) NumChartTypes() int {
	return len(d.typeMap)
}

func (d *DBF) ResolveChartTypes(query string) []string {
	if query == "" {
		return nil
	}

	queryUpper := toUpper(query)

	if ct, ok := d.typeMap[query]; ok {
		return []string{ct.Code}
	}

	if ct, ok := d.typeMap[queryUpper]; ok {
		return []string{ct.Code}
	}

	var codes []string
	for _, ct := range d.typeMap {
		if contains(toUpper(ct.Type), queryUpper) || contains(toUpper(ct.Category), queryUpper) {
			codes = append(codes, ct.Code)
		}
	}
	return codes
}

func trim(s string) string {
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}

func toUpper(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (d *DBF) parseAirportByRow(rowNum int) *Airport {
	getField := func(name string) string {
		return trim(d.airports.FieldValueByName(rowNum, name))
	}

	icao := getField("F5_6")
	if icao == "" {
		return nil
	}

	return &Airport{
		ICAO:              icao,
		IATA:              getField("F5_107"),
		Name:              getField("ARPT_NAME"),
		City:              getField("CITY"),
		State:             getField("ST_PR"),
		CountryCode:       getField("J5_3"),
		Latitude:          parseCoordinate(getField("F5_36")),
		Longitude:         parseCoordinate(getField("F5_37")),
		MagneticVariation: parseMagneticVar(getField("F5_39")),
		LongestRunwayFt:   parseInt(getField("F5_54")),
		Timezone:          getField("J5_15"),
		AirportUse:        parseAirportUse(getField("J5_14")),
		Customs:           parseCustoms(getField("J5_39")),
		Beacon:            getField("J5_8") == "Y",
		JetStartUnit:      getField("J5_10") == "Y",
		Oxygen:            parseOxygen(getField("J5_6")),
		RepairTypes:       parseRepairTypes(getField("J5_7")),
		FuelTypes:         parseFuelTypes(getField("J5_5")),
	}
}

func parseCoordinate(s string) float64 {
	if len(s) < 9 {
		return 0
	}

	var dir byte
	if s[0] == 'N' || s[0] == 'S' || s[0] == 'E' || s[0] == 'W' {
		dir = s[0]
		s = s[1:]
	}

	var degLen int
	if dir == 'N' || dir == 'S' {
		degLen = 2
	} else {
		degLen = 3
	}

	if len(s) < degLen+2 {
		return 0
	}

	var deg, min float64
	var sec float64

	fmt.Sscanf(s[:degLen], "%f", &deg)
	fmt.Sscanf(s[degLen:degLen+2], "%f", &min)
	fmt.Sscanf(s[degLen+2:], "%f", &sec)

	result := deg + min/60 + sec/3600

	if dir == 'S' || dir == 'W' {
		result = -result
	}

	return result
}

func parseMagneticVar(s string) float64 {
	if len(s) < 2 {
		return 0
	}

	var dir byte
	if s[0] == 'E' || s[0] == 'W' {
		dir = s[0]
		s = s[1:]
	}

	var val float64
	fmt.Sscanf(s, "%f", &val)

	if dir == 'W' {
		val = -val
	}

	return val
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseOxygen(s string) []string {
	if len(s) < 5 {
		return nil
	}
	var types []string
	oxygenMap := []string{
		"Unspecified",
		"Low Pressure",
		"High Pressure",
		"High Low Pressure Bottle",
		"High Pressure Bottle",
	}
	for i := 0; i < 5 && i < len(s); i++ {
		if s[i] == 'Y' && i < len(oxygenMap) {
			types = append(types, oxygenMap[i])
		}
	}
	return types
}

func parseRepairTypes(s string) []string {
	if len(s) < 4 {
		return nil
	}
	var types []string
	repairMap := []string{
		"Minor Airframe",
		"Minor Engine",
		"Major Airframe",
		"Major Engine",
	}
	for i := 0; i < 4 && i < len(s); i++ {
		if s[i] == 'Y' && i < len(repairMap) {
			types = append(types, repairMap[i])
		}
	}
	return types
}

func parseFuelTypes(s string) []string {
	if len(s) == 0 {
		return nil
	}
	var types []string
	fuelMap := []string{
		"Unspecified",
		"73 Octane",
		"80-87 Octane",
		"100 Octane (LL)",
		"100-130 Octane",
		"115-145 Octane",
		"Mogas",
		"Jet",
		"Jet A",
		"Jet A-1",
		"Jet A+",
		"Jet B",
		"Jet 4",
		"Jet 5",
	}
	for i := 0; i < len(s) && i < len(fuelMap); i++ {
		if s[i] == 'Y' {
			types = append(types, fuelMap[i])
		}
	}
	return types
}

func parseAirportUse(s string) string {
	switch s {
	case "P":
		return "Private"
	case "M":
		return "Military"
	case "J":
		return "Joint"
	case "Y":
		return "Public"
	default:
		return "Unknown"
	}
}

func parseCustoms(s string) string {
	switch s {
	case "A":
		return "ADCUS"
	case "R":
		return "On Restricted Basis"
	case "P":
		return "On Prior Request"
	case "Y":
		return "Yes"
	default:
		return "No"
	}
}

func (d *DBF) GetAirport(icao string) *Airport {
	return d.airportMap[icao]
}

func (d *DBF) GetAllAirports() []*Airport {
	airports := make([]*Airport, 0, len(d.airportMap))
	for _, a := range d.airportMap {
		airports = append(airports, a)
	}
	return airports
}
