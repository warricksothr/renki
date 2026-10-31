package goal

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestEmptySearchesAdvanceBearingsWithoutClaimingCombatOrResettingBudgets(t *testing.T) {
	host, r, v, id := fixture(t)
	v.Perception.Characters = nil
	order, err := r.Next(v, nil)
	for i, sector := range []int{0, 4, 2, 6, 1, 5} {
		if err != nil || order == nil {
			t.Fatalf("sector %d: %v %v", sector, order, err)
		}
		args := order.Args.(*protocol.AttackArgs)
		if args.SearchCenter == nil || areaSector(protocol.Vec3{}, *args.SearchCenter) != sector || args.SearchSector != nil || args.Entity != "" || !args.Kite {
			t.Fatal(args)
		}
		g, _ := host.Store.Get(id)
		if g.Pending.Search == nil || g.Pending.Search.Sector != sector || len(g.Searches) != i {
			t.Fatal("search intent was not checkpointed", g)
		}
		taskID := fmt.Sprintf("search-%d", i)
		if err := r.Dispatched(taskID); err != nil {
			t.Fatal(err)
		}
		fresh(&v, 30)
		order, err = r.Next(v, &behaviour.Result{TaskID: taskID, Status: protocol.StatusFailed, Reason: protocol.ReasonInvalidTarget, Detail: emptySectorDetail})
	}
	if order != nil || err != nil {
		t.Fatal("exceeded task budget", order, err)
	}
	g, _ := host.Store.Get(id)
	if g.Status != "needs_plan" || g.Proof != nil || g.Repeated != 0 || g.Attempts != 6 || g.TaskCount != 6 || g.WaitS != 0 || len(g.Searches) != 6 {
		t.Fatal(g)
	}
	for _, search := range g.Searches {
		if !search.Empty || search.Outcome.TaskID == "" {
			t.Fatal(search)
		}
	}
	if err := host.Queue(id); err == nil {
		t.Fatal("resume reset exhausted search budget")
	}
}

func TestSearchLedgerSurvivesExplicitResumeAndUncertainNextIntentCannotReplay(t *testing.T) {
	host, r, v, id := fixture(t)
	v.Perception.Characters = nil
	attack(t, r, v, "search-1")
	// Readiness wait leaves a completed checkpoint with no next native intent.
	v.State.Stamina = 30
	_, err := r.Next(v, &behaviour.Result{TaskID: "search-1", Status: protocol.StatusFailed, Reason: protocol.ReasonInvalidTarget, Detail: emptySectorDetail})
	var wait *behaviour.Delay
	if !errors.As(err, &wait) {
		t.Fatal(err)
	}
	reopened, err := Open(host.Store.path)
	if err != nil {
		t.Fatal(err)
	}
	host = &Template{Store: reopened}
	g, _ := reopened.Get(id)
	if g.Status != "suspended" || len(g.Searches) != 1 || !g.Searches[0].Empty || g.Attempts != 1 {
		t.Fatal(g)
	}
	if err := host.Queue(id); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(params{id})
	rr, err := host.Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	r = rr.(*runner)
	v.State.Stamina = 60
	fresh(&v, 10)
	order, err := r.Next(v, nil)
	if err != nil || order == nil || order.Args.(*protocol.AttackArgs).SearchCenter == nil || areaSector(protocol.Vec3{}, *order.Args.(*protocol.AttackArgs).SearchCenter) != 4 {
		t.Fatal(order, err)
	}
	if err := r.Dispatched("search-2"); err != nil {
		t.Fatal(err)
	}
	reopened, err = Open(host.Store.path)
	if err != nil {
		t.Fatal(err)
	}
	g, _ = reopened.Get(id)
	if g.Status != "needs_plan" || g.Pending.Search.Sector != 4 || g.Pending.TaskID != "search-2" || len(g.Searches) != 1 || g.Attempts != 2 || g.WaitS != 10 {
		t.Fatal(g)
	}
	if err := (&Template{Store: reopened}).Queue(id); err == nil {
		t.Fatal("replayed uncertain search")
	}
}

func TestIncompleteSearchesNeverBecomeEmptyCoverageAndRetainFailureLimit(t *testing.T) {
	for _, detail := range []string{"no completed scouting route; creature search has no coverage", "cannot return along a checked scouting route"} {
		t.Run(detail, func(t *testing.T) {
			host, r, v, id := fixture(t)
			v.Perception.Characters = nil
			attack(t, r, v, "search-0")
			for i := 0; i < 3; i++ {
				order, err := r.Next(v, &behaviour.Result{TaskID: fmt.Sprintf("search-%d", i), Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable, Detail: detail})
				if err != nil {
					t.Fatal(err)
				}
				if i < 2 {
					if order == nil {
						t.Fatal("did not try a fresh bounded bearing")
					}
					if err := r.Dispatched(fmt.Sprintf("search-%d", i+1)); err != nil {
						t.Fatal(err)
					}
				} else if order != nil {
					t.Fatal("escaped repeated failure limit")
				}
			}
			g, _ := host.Store.Get(id)
			if g.Status != "needs_plan" || g.Attempts != 3 || len(g.Searches) != 3 || g.Proof != nil {
				t.Fatal(g)
			}
			for _, search := range g.Searches {
				if search.Empty {
					t.Fatal("incomplete search claimed empty coverage", search)
				}
			}
		})
	}
}

func TestSearchReceiptAndNextIntentCheckpointAreAtomic(t *testing.T) {
	host, r, v, id := fixture(t)
	v.Perception.Characters = nil
	attack(t, r, v, "search-1")
	write := host.Store.write
	writes := 0
	host.Store.write = func(raw []byte) error {
		writes++
		if writes == 2 {
			return errors.New("disk full before next intent")
		}
		return write(raw)
	}
	order, err := r.Next(v, &behaviour.Result{TaskID: "search-1", Status: protocol.StatusFailed, Reason: protocol.ReasonInvalidTarget, Detail: emptySectorDetail})
	if err == nil || order != nil {
		t.Fatal("dispatch escaped failed intent checkpoint", order, err)
	}
	g, _ := host.Store.Get(id)
	if len(g.Searches) != 1 || !g.Searches[0].Empty || g.Pending != nil || g.Attempts != 1 {
		t.Fatal(g)
	}
}

func TestPolicyCannotChooseItsOwnSectorOrEnlargeSearchRadius(t *testing.T) {
	for _, p := range []string{`{"world":"one","searchRadius":48,"searchSector":4}`, `{"world":"one","searchRadius":60,"searchSector":0}`, `{"world":"one","searchRadius":48,"searchSector":0,"searchOffset":60}`} {
		host, r, v, id := fixture(t)
		v.Perception.Characters = nil
		source := "def step(context, state, outcome):\n return {\"action\": \"run\", \"template\": \"combat\", \"params\": " + p + "}\n"
		if err := host.Store.update(id, func(g *Record) error { g.Policy = source; g.PolicyHash = hash(source); return nil }); err != nil {
			t.Fatal(err)
		}
		order, err := r.Next(v, nil)
		g, _ := host.Store.Get(id)
		if err != nil || order != nil || g.Status != "needs_plan" || g.Attempts != 0 || len(g.Searches) != 0 {
			t.Fatal(order, err, g)
		}
	}
}

func TestAreaSearchesStayAnchoredAndPersistWithoutLeavingTheGoalEnvelope(t *testing.T) {
	for _, radius := range []float64{100, 200} {
		t.Run(fmt.Sprint(radius), func(t *testing.T) {
			host, r, v, id := fixture(t)
			if err := host.Store.update(id, func(g *Record) error { g.Radius = radius; return nil }); err != nil {
				t.Fatal(err)
			}
			v.Perception.Characters = nil
			order, err := r.Next(v, nil)
			seen := map[protocol.Vec3]bool{}
			for i := 0; i < 6; i++ {
				if err != nil || order == nil {
					t.Fatal(order, err)
				}
				args := order.Args.(*protocol.AttackArgs)
				if args.SearchCenter == nil || seen[*args.SearchCenter] || args.SearchOffset != nil || args.SearchSector != nil {
					t.Fatal(args)
				}
				seen[*args.SearchCenter] = true
				g, _ := host.Store.Get(id)
				if distance(*args.SearchCenter, g.Center)+*args.SearchRadius+52 > radius+1e-6 {
					t.Fatal("area escaped scope", args)
				}
				if g.Pending.Search.Center == nil || *g.Pending.Search.Center != *args.SearchCenter {
					t.Fatal("area not checkpointed")
				}
				task := fmt.Sprintf("area-%d", i)
				if err := r.Dispatched(task); err != nil {
					t.Fatal(err)
				}
				// Native local searches finish around their selected area, rather than
				// resetting the next task's search anchor to a drifting return point.
				v.State.Pos = *args.SearchCenter
				order, err = r.Next(v, &behaviour.Result{TaskID: task, Status: protocol.StatusFailed, Reason: protocol.ReasonInvalidTarget, Detail: emptyAreaDetail})
			}
			reopened, err := Open(host.Store.path)
			if err != nil {
				t.Fatal(err)
			}
			g, _ := reopened.Get(id)
			if g.Attempts != 6 || len(g.Searches) != 6 || g.Proof != nil {
				t.Fatal(g)
			}
			for _, search := range g.Searches {
				if !search.Empty || search.Plan.Center == nil {
					t.Fatal(search)
				}
			}
			original := *g.Searches[5].Plan.Center
			(*g.Searches[5].Plan.Center)[0] += 1000
			if err := validateRecord(g); err == nil {
				t.Fatal("out of bounds area accepted")
			}
			g, _ = reopened.Get(id)
			if *g.Searches[5].Plan.Center != original {
				t.Fatal("snapshot aliased store")
			}
		})
	}
}
