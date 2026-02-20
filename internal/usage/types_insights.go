package usage

import "time"

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

type ProductivityPerHourBreakdown map[time.Time]ProductivityScore
