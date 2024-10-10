package game

type Canton struct {
	ID           int    // Unique identifier for the canton
	Name         string // Full name of the canton
	Abbreviation string // Abbreviation
	AdjacentIDs  []int  // IDs of adjacent cantons
}

// Map represents the game map, containing all the cantons.
type Map struct {
	Cantons map[int]*Canton // Maps canton IDs to Canton pointers
}

// NewMap creates and returns a new Map instance.
func NewMap() *Map {
	return &Map{
		Cantons: make(map[int]*Canton),
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

// contains checks if a slice contains a specific item. (avoid duplicate ownership etc..)
func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// initialize the map with cantons and their adjacents
func CreateMap() *Map {
	m := NewMap()

	// Initialize cantons
	for id, abbrev := range cantonAbbreviations {
		canton := &Canton{
			ID:           id,
			Name:         cantonNames[id],
			Abbreviation: abbrev,
			AdjacentIDs:  []int{},
		}
		m.AddCanton(canton)
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

// GLOBAL DATA. I didn't actually have access to the paper or the map, so I just came up with this basic representation of the map. This whole part needs to be replaced with the actual map data.

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
