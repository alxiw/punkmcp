package model

// Beer represents a complete beer recipe returned by the PunkAPI v3 /api/v3/beers endpoint.
// Fields that can be null in the BrewDog DIY Dog dataset are represented as pointers.
type Beer struct {
	ID               int         `json:"id"`
	Name             string      `json:"name"`
	Tagline          string      `json:"tagline"`
	FirstBrewed      string      `json:"first_brewed"`
	Description      string      `json:"description"`
	Image            *string     `json:"image"`
	ABV              float64     `json:"abv"`
	IBU              *float64    `json:"ibu"`
	TargetFG         *float64    `json:"target_fg"`
	TargetOG         *float64    `json:"target_og"`
	EBC              *float64    `json:"ebc"`
	SRM              *float64    `json:"srm"`
	PH               *float64    `json:"ph"`
	AttenuationLevel *float64    `json:"attenuation_level"`
	Volume           Volume      `json:"volume"`
	BoilVolume       Volume      `json:"boil_volume"`
	Method           Method      `json:"method"`
	Ingredients      Ingredients `json:"ingredients"`
	FoodPairing      []string    `json:"food_pairing"`
	BrewersTips      string      `json:"brewers_tips"`
	ContributedBy    string      `json:"contributed_by"`
}

// Volume represents a liquid volume measurement (e.g., 20 litres).
type Volume struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// Method describes the brewing process: mashing schedule, fermentation temperature, and optional twist.
type Method struct {
	MashTemp     []MashTemp   `json:"mash_temp"`
	Fermentation Fermentation `json:"fermentation"`
	Twist        *string      `json:"twist"`
}

// MashTemp represents a single mash temperature step and optional duration in minutes.
type MashTemp struct {
	Temp     Temperature `json:"temp"`
	Duration *int        `json:"duration"`
}

// Fermentation describes the fermentation temperature for a beer recipe.
type Fermentation struct {
	Temp Temperature `json:"temp"`
}

// Temperature represents a temperature measurement where the numeric value may be null.
type Temperature struct {
	Value *float64 `json:"value"`
	Unit  string   `json:"unit"`
}

// Ingredients contains the malts, hops, and yeast used in a beer recipe.
type Ingredients struct {
	Malt  []Malt  `json:"malt"`
	Hops  []Hop   `json:"hops"`
	Yeast *string `json:"yeast"`
}

// Malt represents a malt ingredient and its quantity.
type Malt struct {
	Name   string `json:"name"`
	Amount Amount `json:"amount"`
}

// Hop represents a hop addition, including timing and brewing attribute.
type Hop struct {
	Name      string  `json:"name"`
	Amount    Amount  `json:"amount"`
	Add       string  `json:"add"`
	Attribute *string `json:"attribute"`
}

// Amount represents an ingredient weight or quantity measurement.
type Amount struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// Clone returns a deep copy of Beer so callers cannot mutate shared slices or pointer fields.
func (b Beer) Clone() Beer {
	out := b
	out.Image = clonePtr(b.Image)
	out.IBU = clonePtr(b.IBU)
	out.TargetFG = clonePtr(b.TargetFG)
	out.TargetOG = clonePtr(b.TargetOG)
	out.EBC = clonePtr(b.EBC)
	out.SRM = clonePtr(b.SRM)
	out.PH = clonePtr(b.PH)
	out.AttenuationLevel = clonePtr(b.AttenuationLevel)

	out.Method.Fermentation.Temp.Value = clonePtr(b.Method.Fermentation.Temp.Value)
	out.Method.Twist = clonePtr(b.Method.Twist)
	if b.Method.MashTemp != nil {
		out.Method.MashTemp = make([]MashTemp, len(b.Method.MashTemp))
		for i, mt := range b.Method.MashTemp {
			out.Method.MashTemp[i] = MashTemp{
				Temp: Temperature{
					Value: clonePtr(mt.Temp.Value),
					Unit:  mt.Temp.Unit,
				},
				Duration: clonePtr(mt.Duration),
			}
		}
	}

	out.Ingredients.Yeast = clonePtr(b.Ingredients.Yeast)
	if b.Ingredients.Malt != nil {
		out.Ingredients.Malt = make([]Malt, len(b.Ingredients.Malt))
		copy(out.Ingredients.Malt, b.Ingredients.Malt)
	}
	if b.Ingredients.Hops != nil {
		out.Ingredients.Hops = make([]Hop, len(b.Ingredients.Hops))
		for i, h := range b.Ingredients.Hops {
			out.Ingredients.Hops[i] = Hop{
				Name:      h.Name,
				Amount:    h.Amount,
				Add:       h.Add,
				Attribute: clonePtr(h.Attribute),
			}
		}
	}

	if b.FoodPairing != nil {
		out.FoodPairing = make([]string, len(b.FoodPairing))
		copy(out.FoodPairing, b.FoodPairing)
	}

	return out
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
