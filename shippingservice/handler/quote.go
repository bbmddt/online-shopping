package handler

import (
	"fmt"
	"math"
)

// currency quote type
type Quote struct {
	Dollars uint32
	Cents   uint32
}

// quote
func (q Quote) String() string {
	return fmt.Sprintf("$%d.%d", q.Dollars, q.Cents)
}

// create quote based on the count of items
func CreateQuoteFromCount(count int) Quote {
	return CreateQuoteFromFloat(8.99)
}

// create quote
func CreateQuoteFromFloat(value float64) Quote {
	units, fraction := math.Modf(value)
	return Quote{
		uint32(units),
		uint32(math.Trunc(fraction * 100)),
	}
}
