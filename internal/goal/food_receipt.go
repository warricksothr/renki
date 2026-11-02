package goal

import (
	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
)

func cookingDelivery(rows []*protocol.Trace, id, output, station string) (int, string) {
	return behaviour.CookingDelivery(rows, id, output, station)
}
