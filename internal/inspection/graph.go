package inspection

type Phase struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}
type Edge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Guard string `json:"guard"`
}
type Graph struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Active string  `json:"active,omitempty"`
	Source string  `json:"source"`
	Phases []Phase `json:"phases"`
	Edges  []Edge  `json:"edges"`
}

func peelGraph(active string) *Graph {
	return &Graph{ID: "native-peel-v1", Label: "Native peel", Active: active, Source: "Live native phase; selected possible transitions, not an exhaustive graph or observed history.",
		Phases: []Phase{{"prepare", "Prepare", "Survey fighting ground and escape; require native readiness."},
			{"draw", "Draw", "Try to acquire selected pursuit without entering melee."},
			{"lead", "Lead", "Lead while sampling the entire group; missing members remain uncertain."},
			{"isolate", "Isolate", "Wait for continuous single-pursuer evidence at the selected ground."},
			{"fight", "Fight", "Native melee continuously rechecks isolation and ordinary safety."},
			{"escape", "Escape", "Cancel swings and use bounded native escape."}},
		Edges: []Edge{{"prepare", "draw", "ground and readiness admitted"}, {"draw", "lead", "selected pursuit observed"}, {"lead", "isolate", "at fighting ground"},
			{"isolate", "lead", "continued repositioning"}, {"isolate", "fight", "native isolation and melee admission"}, {"lead", "fight", "at ground with native isolation and melee admission"},
			{"prepare", "escape", "preparation or survey failure"}, {"draw", "escape", "draw budget or safety failure"},
			{"lead", "escape", "route, convergence or budget failure"}, {"isolate", "escape", "lost evidence or safety"}, {"fight", "escape", "isolation lost or native refusal"}}}
}

func foodGraph(active string) *Graph {
	return &Graph{ID: "food-adapter-v1", Label: "Food host adapter", Active: active, Source: "Saved host checkpoint; selected transitions only. Eating, recovery and household work can interrupt phases; checkpoints do not prove body ownership or live stock.",
		Phases: []Phase{{"home", "Home", "Establish home and initial resupply context."}, {"ready", "Ready", "Inspect candidates and acquire ready meals."},
			{"raw", "Raw food", "Acquire ingredients or forage; reconcile receipts."}, {"wood", "Fuel", "Acquire needed cooking fuel."},
			{"cook", "Cook", "Accepted checkpoint value; current host dispatches cooking from Fuel without changing the saved phase."}, {"recover", "Recover", "Accepted checkpoint value; current host checks eating and recovery across phases."},
			{"complete", "Complete", "Host checks reserve/meal/health and optional household deposit evidence."}},
		Edges: []Edge{{"home", "ready", "home context admitted"}, {"ready", "raw", "ready meal shortfall"}, {"raw", "wood", "ingredient available; inspect cooking fuel"},
			{"wood", "raw", "cooking receipt reconciled"}, {"cook", "raw", "cooking receipt reconciled from accepted checkpoint"},
			{"ready", "complete", "host reserve, health and objective verified"}, {"raw", "complete", "host reserve, health and objective verified"},
			{"wood", "complete", "host reserve, health and objective verified"}}}
}
