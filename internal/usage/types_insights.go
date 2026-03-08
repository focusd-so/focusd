package usage

type DayInsights struct {
	ProductivityScore            ProductivityScore
	ProductivityPerHourBreakdown ProductivityPerHourBreakdown
	LLMDailySummary              *LLMDailySummary
	TopDistractions              []DistractionBreakdown
	TopBlocked                   []BlockedBreakdown
	ProjectBreakdown             []ProjectBreakdown
	CommunicationBreakdown       []CommunicationBreakdown
}

type DistractionBreakdown struct {
	Name    string `json:"name"`
	Minutes int    `json:"minutes"`
}

type BlockedBreakdown struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ProjectBreakdown struct {
	Name    string `json:"name"`
	Minutes int    `json:"minutes"`
}

type CommunicationBreakdown struct {
	Name    string `json:"name"`
	Minutes int    `json:"minutes"`
}

type ProductivityScore struct {
	ProductiveSeconds  int
	DistractiveSeconds int
	OtherSeconds       int
	ProductivityScore  int
}

func (p *ProductivityScore) addSeconds(classification Classification, seconds int) {
	switch classification {
	case ClassificationProductive:
		p.ProductiveSeconds += seconds
	case ClassificationDistracting:
		p.DistractiveSeconds += seconds
	default:
		p.OtherSeconds += seconds
	}
}

type ProductivityPerHourBreakdown map[int]ProductivityScore
