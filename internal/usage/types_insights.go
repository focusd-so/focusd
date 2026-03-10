package usage

type DayInsights struct {
	ProductivityScore            ProductivityScore                 `json:"productivity_score"`
	ProductivityPerHourBreakdown ProductivityPerHourBreakdown      `json:"productivity_per_hour_breakdown"`
	LLMDailySummary              *LLMDailySummary                  `json:"llm_daily_summary"`
	TopDistractions              []DistractionBreakdown            `json:"top_distractions"`
	TopBlocked                   []BlockedBreakdown                `json:"top_blocked"`
	ProjectBreakdown             []ProjectBreakdown                `json:"project_breakdown"`
	CommunicationBreakdown       map[string]CommunicationBreakdown `json:"communication_breakdown"`
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
	Channel string `json:"channel"`
	Minutes int    `json:"minutes"`
}

type ProductivityScore struct {
	ProductiveSeconds  int `json:"productive_seconds"`
	DistractiveSeconds int `json:"distractive_seconds"`
	IdleSeconds        int `json:"idle_seconds"`
	OtherSeconds       int `json:"other_seconds"`
	ProductivityScore  int `json:"productivity_score"`
}

func (p *ProductivityScore) addSeconds(classification Classification, seconds int, isIdle bool) {
	if isIdle {
		p.IdleSeconds += seconds
		return
	}
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
