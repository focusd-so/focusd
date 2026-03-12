package usage

type DayInsights struct {
	ProductivityScore            ProductivityScore                 `json:"productivity_score"`
	ProductivityPerHourBreakdown ProductivityPerHourBreakdown      `json:"productivity_per_hour_breakdown"`
	LLMDailySummary              *LLMDailySummary                  `json:"llm_daily_summary"`
	TopDistractions              map[string]int                    `json:"top_distractions"`
	TopBlocked                   map[string]int                    `json:"top_blocked"`
	ProjectBreakdown             map[string]int                    `json:"project_breakdown"`
	CommunicationBreakdown       map[string]CommunicationBreakdown `json:"communication_breakdown"`
}

type ProjectBreakdown struct {
	Name            string `json:"name"`
	DurationSeconds int    `json:"duration_seconds"`
}

type CommunicationBreakdown struct {
	Name            string `json:"name"`
	Channel         string `json:"channel"`
	DurationSeconds int    `json:"duration_seconds"`
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
