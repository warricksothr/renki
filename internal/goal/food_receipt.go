package goal

import (
	"math"

	"github.com/warricksothr/renki/internal/protocol"
)

// cookingDelivery checks native receipts, not task success text. Missing data
// stays missing; a zero count never substitutes for an absent observation.
func cookingDelivery(rows []*protocol.Trace, id, output, station string) (int, string) {
	var final, request map[string]any
	outputs, picked := map[string]int{}, map[string]int{}
	for _, t := range rows {
		if t == nil || t.Task != id || t.Source != "cooking" || t.Data["id"] != id || t.Data["station"] != station {
			continue
		}
		d := t.Data
		switch t.Msg {
		case "normal cooking removal request observed":
			if request != nil {
				return 0, ""
			}
			request = d
		case "verified cooked inventory receipt":
			if final != nil {
				return 0, ""
			}
			final = d
		case "normal station output spawned", "correlated remote station output observed":
			e, _ := d["entity"].(string)
			n := receiptInt(d["count"])
			if e == "" || n < 1 || d["output"] != output || outputs[e] > 0 {
				return 0, ""
			}
			outputs[e] = n
		case "verified cooking drop pickup":
			e, _ := d["entity"].(string)
			n := receiptInt(d["count"])
			if e == "" || n < 1 || d["output"] != output {
				return 0, ""
			}
			picked[e] += n
		}
	}
	if final == nil || request == nil || final["output"] != output {
		return 0, ""
	}
	mode, _ := final["evidence"].(string)
	n := receiptInt(final["count"])
	if n < 1 || n > 10 || request["evidence"] != mode || receiptInt(request["requestedCount"]) != n || mode != "direct_spawn" && mode != "remote_correlation" || len(outputs) != len(picked) {
		return 0, ""
	}
	total := 0
	for e, c := range outputs {
		if picked[e] != c {
			return 0, ""
		}
		total += c
	}
	if total != n {
		return 0, ""
	}
	return n, mode
}
func receiptInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		if finite(n) && n == math.Trunc(n) && n >= 0 && n <= 100 {
			return int(n)
		}
	}
	return -1
}
