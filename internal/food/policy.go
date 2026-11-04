package food

import (
	"errors"
	"math"
	"slices"
	"strings"

	"github.com/warricksothr/renki/internal/protocol"
)

// Policy uses progression tiers, not the upgrade quality of an inventory item.
// Unknown foods remain reserved until classified, except during an emergency.
type Policy struct {
	ActivityAware bool           `json:"activityAware"`
	IdleMaxTier   int            `json:"idleMaxTier"`
	Tiers         map[string]int `json:"tiers"`
}

func (p Policy) Validate() error {
	if p.IdleMaxTier < 0 || p.IdleMaxTier > 8 || p.ActivityAware && p.IdleMaxTier == 0 || len(p.Tiers) > 256 {
		return errors.New("food requires idleMaxTier 1–8 when enabled and at most 256 tier overrides")
	}
	for name, tier := range p.Tiers {
		if strings.TrimSpace(name) != name || name == "" || len(name) > 128 || tier < 1 || tier > 8 {
			return errors.New("food.tiers requires prefab names and integer tiers 1–8")
		}
	}
	return nil
}

func (p Policy) Tier(prefab string) int {
	if tier, ok := p.Tiers[prefab]; ok {
		return tier
	}
	// Vanilla progression: Meadows, Black Forest, Swamp, Mountains, Plains,
	// Mistlands and Ashlands. Unlisted/modded meals are deliberately unknown.
	groups := []string{
		"Raspberry Mushroom Honey CookedMeat CookedDeerMeat NeckTailGrilled BoarJerky",
		"Blueberries MushroomYellow Carrot CarrotSoup DeerStew MinceMeatSauce QueensJam FishCooked",
		"TurnipStew Sausages BlackSoup Muckshake",
		"CookedWolfMeat WolfMeatSkewer WolfJerky Eyescream Onion OnionSoup",
		"Bread FishWraps LoxPie CookedLoxMeat BloodPudding SerpentStew CookedSerpentMeat",
		"CookedHareMeat MisthareSupreme MeatPlatter Salad MushroomOmelette FishAndBread MushroomMagecap MushroomJotunPuffs YggdrasilPorridge SeekerAspic StuffedMushroom",
		"CookedAsksvinMeat CookedVoltureMeat MashedMeat FierySvinstew PiquantPie RoastedCrustPie ScorchingMedley SizzlingBerryBroth MarinatedGreens SparklingShroomshake",
	}
	for i, group := range groups {
		if slices.Contains(strings.Fields(group), prefab) {
			return i + 1
		}
	}
	return 0
}

type Candidate struct {
	Prefab string `json:"prefab"`
	Tier   int    `json:"tier"` // zero means unknown
	Reason string `json:"reason"`
}

// Decision is a pure explanation used by execution, durable goals and readers.
type Decision struct {
	Version    int         `json:"version"`
	Activity   string      `json:"activity"`
	Mode       string      `json:"mode"`
	Reason     string      `json:"reason"`
	Item       string      `json:"item,omitempty"`
	Candidates []Candidate `json:"candidates"`
}

func WorkingTask(task string) bool {
	switch task {
	case protocol.TaskMoveTo, protocol.TaskFollow, protocol.TaskEscort, protocol.TaskHarvest,
		protocol.TaskAttack, protocol.TaskFlee, protocol.TaskDeposit, protocol.TaskWithdraw:
		return true
	}
	return false
}

// Decide never changes slots or bypasses the game's CanEat. Idle refreshes wait
// until the final two minutes; filling an empty slot still takes precedence.
func Decide(st *protocol.State, p Policy, activity string) Decision {
	d := Decision{Version: 1, Activity: activity, Mode: "idle", Reason: "no_eligible_idle_meal", Candidates: []Candidate{}}
	if st == nil || st.Food == nil || st.Phase != protocol.PhaseInWorld || st.HP <= 0 {
		d.Reason = "no_live_food_observation"
		return d
	}
	if activity == "" && st.Task != nil && !protocol.Final(st.Task.Status) && WorkingTask(st.Task.Task) {
		d.Activity = "task:" + st.Task.Task
	}
	if !p.ActivityAware {
		d.Mode = "legacy"
	} else if len(st.Food.Active) == 0 || st.HPMax > 0 && st.HP < st.HPMax*.6 || st.HP < 35 {
		d.Mode = "emergency"
	} else if d.Activity != "" && !strings.HasPrefix(d.Activity, "idle:") {
		d.Mode = "work"
	}
	active := map[string]float64{}
	for _, f := range st.Food.Active {
		active[f.Prefab] = f.RemainingS
	}
	var best *protocol.CarriedFood
	for i := range st.Food.Carried {
		f := &st.Food.Carried[i]
		tier := p.Tier(f.Prefab)
		reason := "eligible"
		_, refresh := active[f.Prefab]
		switch {
		case !f.CanEat || f.Prefab == "" || f.Health <= 0 || math.IsNaN(f.Health) || math.IsInf(f.Health, 0) || math.IsNaN(f.Regen) || math.IsInf(f.Regen, 0) || f.Stamina < 0 || math.IsNaN(f.Stamina) || math.IsInf(f.Stamina, 0):
			reason = "native_ineligible"
		case d.Mode == "idle" && (tier == 0 || tier > p.IdleMaxTier):
			reason = "reserved_for_work"
		case d.Mode == "idle" && refresh && active[f.Prefab] > 120:
			reason = "refresh_not_needed"
		}
		d.Candidates = append(d.Candidates, Candidate{Prefab: f.Prefab, Tier: tier, Reason: reason})
		if reason != "eligible" {
			continue
		}
		better := best == nil
		if best != nil {
			_, oldRefresh := active[best.Prefab]
			switch {
			case len(st.Food.Active) < 3 && refresh != oldRefresh:
				better = !refresh
			case d.Mode == "idle" && tier != p.Tier(best.Prefab):
				better = tier < p.Tier(best.Prefab)
			case d.Mode == "work" && f.Health+f.Stamina != best.Health+best.Stamina:
				better = f.Health+f.Stamina > best.Health+best.Stamina
			default:
				better = f.Health > best.Health || f.Health == best.Health && (f.Regen > best.Regen || f.Regen == best.Regen && f.Prefab < best.Prefab)
			}
		}
		if better {
			best = f
		}
	}
	slices.SortFunc(d.Candidates, func(a, b Candidate) int { return strings.Compare(a.Prefab, b.Prefab) })
	if best != nil {
		d.Item, d.Reason = best.Prefab, d.Mode+"_eligible_meal"
		for i := range d.Candidates {
			if d.Candidates[i].Prefab == d.Item {
				d.Candidates[i].Reason = "selected"
			}
		}
	} else if d.Mode != "idle" {
		d.Reason = "no_eligible_meal"
	}
	return d
}
