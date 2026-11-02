package goal

import (
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestFoodSourceRejectsCapturedHostileCorridorAndSelectsAnotherYield(t *testing.T) {
	_, r, v, _ := acquireFixture(t)
	occupied := forageResource("occupied", "Raspberry", protocol.Vec3{.5, 33, -19})
	clear := forageResource("clear", "Raspberry", protocol.Vec3{-35, 33, -15})
	v.Perception.Resources = []protocol.PerceivedResource{occupied, clear}
	v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "greydwarf", Kind: protocol.CharacterCreature,
		Prefab: "Greydwarf", HPFrac: 1, Hostile: true, Pos: protocol.Vec3{16.35, 32.76, -15.68}}}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskInteract {
		t.Fatal(o, err)
	}
	if a := o.Args.(*protocol.InteractArgs); a.Entity != clear.ID {
		t.Fatalf("selected occupied source: %+v", a)
	}
}

func TestFoodSourceUsesNativeHorizontalHostileClearance(t *testing.T) {
	cases := []struct {
		name, prefab string
		stars        int
		gap          float64
		safe         bool
	}{
		{"neck blocked", "Neck", 0, 15.9, false}, {"neck clear", "Neck", 0, 16.1, true},
		{"boar blocked", "Boar", 0, 23.9, false}, {"boar clear", "Boar", 0, 24.1, true},
		{"greyling blocked", "Greyling", 0, 39.9, false}, {"greyling clear", "Greyling", 0, 40.1, true},
		{"starred neck", "Neck", 1, 20, false}, {"starred boar", "Boar", 1, 30, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, v, _ := acquireFixture(t)
			v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "enemy", Kind: protocol.CharacterCreature,
				Prefab: tc.prefab, Stars: tc.stars, HPFrac: 1, Hostile: true, Pos: protocol.Vec3{tc.gap, 100, 0}}}
			if got := foodSiteSafe(v, protocol.Vec3{}, ""); got != tc.safe {
				t.Fatalf("safe=%v want %v", got, tc.safe)
			}
		})
	}
}

func TestFoodSourceRetainsExactPreyExclusionAndPlayerPassage(t *testing.T) {
	_, _, v, _ := acquireFixture(t)
	v.Perception.Characters = []protocol.PerceivedCharacter{
		{ID: "prey", Kind: protocol.CharacterCreature, Prefab: "Neck", HPFrac: 1, Hostile: true},
		{ID: "owner", Kind: protocol.CharacterPlayer, HPFrac: 1},
	}
	if !foodSiteSafe(v, protocol.Vec3{}, "prey") {
		t.Fatal("exact isolated prey or player blocked")
	}
	v.Perception.Characters = append(v.Perception.Characters, protocol.PerceivedCharacter{
		ID: "other", Kind: protocol.CharacterCreature, Prefab: "Boar", HPFrac: 1, Hostile: true, Pos: protocol.Vec3{20, 0, 0},
	})
	if foodSiteSafe(v, protocol.Vec3{}, "prey") {
		t.Fatal("second creature admitted")
	}
}
