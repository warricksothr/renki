package atlas

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

type Coverage struct {
	Total      int `json:"total"`
	Measured   int `json:"measured"`
	New        int `json:"new"`
	Refreshed  int `json:"refreshed"`
	Unknown    int `json:"unknown"`
	Stale      int `json:"stale"`
	MeasuredM2 int `json:"measuredM2"`
}
type Sample struct {
	Pos      protocol.Vec3 `json:"pos"`
	Water    float64       `json:"water"`
	Observed time.Time     `json:"observed"`
}

// SurveyCoverage counts canonical 4 m sample positions, not visible objects or
// safely traversable ground. First receipt is immutable across confirmations.
// Evicted cells reinserted later have a new first receipt.
func (s *Store) SurveyCoverage(ctx context.Context, key, source string, center protocol.Vec3, radius float64, since, now time.Time) (Coverage, []Sample, error) {
	out := Coverage{}
	if len(key) != 64 || source == "" || len(source) > 128 || math.IsNaN(radius) || math.IsInf(radius, 0) || radius < 1 || radius > 250 || since.IsZero() || now.Before(since) {
		return out, nil, errors.New("invalid measured-coverage scope")
	}
	for _, v := range center {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 1e6 {
			return out, nil, errors.New("invalid measured-coverage center")
		}
	}
	minX, maxX := int(math.Ceil((center[0]-radius)/4)), int(math.Floor((center[0]+radius)/4))
	minZ, maxZ := int(math.Ceil((center[2]-radius)/4)), int(math.Floor((center[2]+radius)/4))
	for x := minX; x <= maxX; x++ {
		for z := minZ; z <= maxZ; z++ {
			if math.Hypot(float64(x)*4-center[0], float64(z)*4-center[2]) <= radius {
				out.Total++
			}
		}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT x,z,h,water,observed,first_received,first_source FROM samples WHERE world=? AND source=? AND x BETWEEN ? AND ? AND z BETWEEN ? AND ? AND observed>=? AND observed<=? LIMIT 20000`, key, source, minX, maxX, minZ, maxZ, now.AddDate(0, 0, -s.policy.KeepDays).UnixMilli(), now.UnixMilli())
	if err != nil {
		return out, nil, err
	}
	defer rows.Close()
	points := []Sample{}
	for rows.Next() {
		var x, z int
		var h, water float64
		var observed, first int64
		var firstSource string
		if err = rows.Scan(&x, &z, &h, &water, &observed, &first, &firstSource); err != nil {
			return out, nil, err
		}
		pos := protocol.Vec3{float64(x) * 4, h, float64(z) * 4}
		if pos.FlatDistance(center) > radius {
			continue
		}
		at := time.UnixMilli(observed)
		out.Measured++
		if at.Before(now.AddDate(0, 0, -s.policy.StaleDays)) {
			out.Stale++
		}
		if !at.Before(since) {
			out.Refreshed++
			if first >= since.UnixMilli() && firstSource == source {
				out.New++
			}
		}
		points = append(points, Sample{Pos: pos, Water: water, Observed: at})
	}
	out.Unknown = out.Total - out.Measured
	out.MeasuredM2 = out.Measured * 16
	return out, points, rows.Err()
}
