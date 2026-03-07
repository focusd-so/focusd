package usage

type DayInsights struct {
	ProductivityScore            ProductivityScore
	ProductivityPerHourBreakdown ProductivityPerHourBreakdown
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
