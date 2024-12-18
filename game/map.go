package game

type Canton struct {
	ID           int    // Unique identifier for the canton
	Name         string // Full name of the canton
	Abbreviation string // Abbreviation
	AdjacentIDs  []int  // IDs of adjacent cantons
	RegionID     int    // ID of the region the canton belongs to
}

type Region struct {
	ID        int    // Unique identifier for the region
	Name      string // Name of the region
	CantonIDs []int  // IDs of cantons in this region
	Bonus     int    // Add bonus troops per turn for controlling this region
}

// Map represents the game map, containing all the cantons and regions.
type Map struct {
	Cantons map[int]*Canton // Maps canton IDs to Canton pointers
	Regions map[int]*Region // Maps region IDs to Region pointers
}

// NewMap creates and returns a new Map instance.
func NewMap() *Map {
	return &Map{
		Cantons: make(map[int]*Canton),
		Regions: make(map[int]*Region),
	}
}

// AddCanton adds a new canton to the map.
func (m *Map) AddCanton(canton *Canton) {
	m.Cantons[canton.ID] = canton
}

// AddBorder adds a bidirectional border between two cantons.
func (m *Map) AddBorder(cantonID1, cantonID2 int) {
	if !contains(m.Cantons[cantonID1].AdjacentIDs, cantonID2) {
		m.Cantons[cantonID1].AdjacentIDs = append(m.Cantons[cantonID1].AdjacentIDs, cantonID2)
	}
	if !contains(m.Cantons[cantonID2].AdjacentIDs, cantonID1) {
		m.Cantons[cantonID2].AdjacentIDs = append(m.Cantons[cantonID2].AdjacentIDs, cantonID1)
	}
}

// contains checks if a slice contains a specific item.
func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// CreateMap initializes the map with cantons, regions, and their adjacents.
func CreateMap() *Map {
	m := NewMap()

	// Initialize regions
	m.Regions = make(map[int]*Region)
	for regionID, regionName := range regionNames {
		m.Regions[regionID] = &Region{
			ID:        regionID,
			Name:      regionName,
			CantonIDs: []int{},
		}
	}

	// Initialize cantons
	for id, abbrev := range cantonAbbreviations {
		regionID := 2 // default to German region
		if val, ok := cantonRegionMap[abbrev]; ok {
			regionID = val
		}
		canton := &Canton{
			ID:           id,
			Name:         cantonNames[id],
			Abbreviation: abbrev,
			AdjacentIDs:  []int{},
			RegionID:     regionID,
		}
		m.AddCanton(canton)
		m.Regions[regionID].CantonIDs = append(m.Regions[regionID].CantonIDs, canton.ID)
	}

	// Add borders between cantons
	for abbrev, neighbors := range adjacencyData {
		id1 := cantonIDMap[abbrev]
		for _, neighborAbbrev := range neighbors {
			id2 := cantonIDMap[neighborAbbrev]
			m.AddBorder(id1, id2)
		}
	}

	return m
}

// REGION DATA
var regionNames = map[int]string{
	1: "French",
	2: "German",
	3: "Italian",
}

// Mapping of canton abbreviations to region IDs
var cantonRegionMap = map[string]int{
	// French-speaking cantons
	"GE": 1,
	"VD": 1,
	"NE": 1,
	"JU": 1,
	"FR": 1,
	"VS": 1,
	// Italian-speaking canton
	"TI": 3,
}

// List of canton abbreviations
var cantonAbbreviations = []string{
	"AG", "AI", "AR", "BE", "BL", "BS", "FR", "GE", "GL", "GR",
	"JU", "LU", "NE", "NW", "OW", "SG", "SH", "SO", "SZ", "TG",
	"TI", "UR", "VD", "VS", "ZG", "ZH",
}

// Corresponding canton names
var cantonNames = []string{
	"Aargau", "Appenzell Innerrhoden", "Appenzell Ausserrhoden", "Bern",
	"Basel-Landschaft", "Basel-Stadt", "Fribourg", "Geneva", "Glarus",
	"Graubünden", "Jura", "Lucerne", "Neuchâtel", "Nidwalden", "Obwalden",
	"St. Gallen", "Schaffhausen", "Solothurn", "Schwyz", "Thurgau",
	"Ticino", "Uri", "Vaud", "Valais", "Zug", "Zürich",
}

// Map of canton abbreviations to their IDs
var cantonIDMap = map[string]int{
	"AG": 0, "AI": 1, "AR": 2, "BE": 3, "BL": 4, "BS": 5,
	"FR": 6, "GE": 7, "GL": 8, "GR": 9, "JU": 10, "LU": 11,
	"NE": 12, "NW": 13, "OW": 14, "SG": 15, "SH": 16, "SO": 17,
	"SZ": 18, "TG": 19, "TI": 20, "UR": 21, "VD": 22, "VS": 23,
	"ZG": 24, "ZH": 25,
}

// Adjacency data: mapping of canton abbreviations to their neighboring cantons
var adjacencyData = map[string][]string{
	"AG": {"BL", "LU", "ZG", "ZH", "SO"},
	"AI": {"AR", "SG"},
	"AR": {"AI", "SG"},
	"BE": {"FR", "JU", "NE", "SO", "VD", "VS", "LU"},
	"BL": {"AG", "BS", "SO", "JU"},
	"BS": {"BL"},
	"FR": {"BE", "VD", "NE"},
	"GE": {"VD"},
	"GL": {"SG", "SZ", "GR"},
	"GR": {"SG", "TI", "GL", "UR"},
	"JU": {"BE", "SO", "BL"},
	"LU": {"AG", "BE", "NW", "OW", "ZG"},
	"NE": {"BE", "FR", "VD"},
	"NW": {"OW", "LU", "UR"},
	"OW": {"NW", "UR", "LU"},
	"SG": {"AI", "AR", "GL", "TG", "ZH", "GR"},
	"SH": {"ZH", "TG"},
	"SO": {"BE", "BL", "JU", "AG"},
	"SZ": {"ZG", "UR", "GL"},
	"TG": {"SH", "SG", "ZH"},
	"TI": {"GR", "VS", "UR"},
	"UR": {"SZ", "OW", "GR", "TI", "NW"},
	"VD": {"GE", "FR", "VS", "NE", "BE"},
	"VS": {"VD", "BE", "TI", "UR"},
	"ZG": {"AG", "SZ", "LU", "ZH"},
	"ZH": {"AG", "SG", "TG", "SH", "ZG"},
}
