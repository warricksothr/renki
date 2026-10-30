package goal

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type Directive struct {
	Action   string          `json:"action"`
	Template string          `json:"template,omitempty"`
	Params   json.RawMessage `json:"params,omitempty"`
	State    json.RawMessage `json:"state,omitempty"`
	WaitS    float64         `json:"waitS,omitempty"`
	Reason   string          `json:"reason,omitempty"`
}

func finite(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }

// Decision trees exclude allocation arithmetic, loops, recursion, comprehensions,
// imports and arbitrary calls. Step limits alone do not bound huge list/string
// multiplication in a single interpreter instruction.
func checkSource(source string) error {
	if len(source) == 0 || len(source) > MaxSource {
		return errors.New("policy source size limit")
	}
	f, err := syntax.Parse("policy.star", source, 0)
	if err != nil {
		return err
	}
	if len(f.Stmts) != 1 {
		return errors.New("policy must define only step(context, state, outcome)")
	}
	def, ok := f.Stmts[0].(*syntax.DefStmt)
	if !ok || def.Name.Name != "step" || len(def.Params) != 3 {
		return errors.New("policy must define step(context, state, outcome)")
	}
	for i, name := range []string{"context", "state", "outcome"} {
		p, ok := def.Params[i].(*syntax.Ident)
		if !ok || p.Name != name {
			return errors.New("invalid step parameters")
		}
	}
	nodes := 0
	syntax.Walk(f, func(n syntax.Node) bool {
		if n == nil || err != nil {
			return false
		}
		nodes++
		if nodes > 2048 {
			err = errors.New("policy syntax node limit")
			return false
		}
		switch x := n.(type) {
		case *syntax.File, *syntax.Ident, *syntax.Literal, *syntax.IfStmt, *syntax.ReturnStmt, *syntax.DictExpr, *syntax.ListExpr, *syntax.ParenExpr, *syntax.CondExpr:
		case *syntax.DictEntry:
			lit, ok := x.Key.(*syntax.Literal)
			if !ok || lit.Token != syntax.STRING {
				err = errors.New("dictionary keys must be string literals")
			}
		case *syntax.IndexExpr:
			if !scalarLiteral(x.Y) {
				err = errors.New("indexes must be scalar literals")
			}
		case *syntax.DefStmt:
			if x != def {
				err = errors.New("nested functions are not allowed")
			}
		case *syntax.AssignStmt:
			if _, ok := x.LHS.(*syntax.Ident); !ok || x.Op != syntax.EQ {
				err = errors.New("only local assignment is allowed")
			}
		case *syntax.BinaryExpr:
			switch x.Op {
			case syntax.AND, syntax.OR:
			case syntax.EQL, syntax.NEQ, syntax.LT, syntax.LE, syntax.GT, syntax.GE:
				if !scalarLiteral(x.X) && !scalarLiteral(x.Y) {
					err = errors.New("comparisons need a scalar literal operand")
				}
			default:
				err = errors.New("allocation/arithmetic operators are not allowed")
			}
		case *syntax.UnaryExpr:
			if x.Op != syntax.NOT && x.Op != syntax.MINUS {
				err = errors.New("unsupported unary operator")
			}
		case *syntax.DotExpr:
			if x.Name.Name != "get" {
				err = errors.New("only dictionary get is allowed")
			}
		case *syntax.CallExpr:
			switch fn := x.Fn.(type) {
			case *syntax.Ident:
				if fn.Name != "len" || len(x.Args) != 1 {
					err = errors.New("only len and dictionary get calls are allowed")
				}
			case *syntax.DotExpr:
				if fn.Name.Name != "get" || len(x.Args) < 1 || len(x.Args) > 2 || !scalarLiteral(x.Args[0]) {
					err = errors.New("invalid dictionary get")
				}
			default:
				err = errors.New("arbitrary calls are not allowed")
			}
		default:
			err = fmt.Errorf("unsupported policy syntax %T", n)
		}
		return err == nil
	})
	return err
}
func policyGlobals(source string) (*starlark.Thread, starlark.StringDict, func(), error) {
	if err := checkSource(source); err != nil {
		return nil, nil, func() {}, err
	}
	t := &starlark.Thread{Name: "goal-policy", Print: func(*starlark.Thread, string) {}}
	t.SetMaxExecutionSteps(20000)
	timer := time.AfterFunc(100*time.Millisecond, func() { t.Cancel("policy time limit") })
	cleanup := func() { timer.Stop() }
	g, err := starlark.ExecFileOptions(&syntax.FileOptions{}, t, "policy.star", source, nil)
	if err == nil {
		g.Freeze()
	}
	return t, g, cleanup, err
}
func ValidatePolicy(source string) error {
	_, _, cleanup, err := policyGlobals(source)
	defer cleanup()
	return err
}
func Evaluate(source string, context any, state json.RawMessage, outcome *Outcome) (Directive, error) {
	var d Directive
	t, g, cleanup, err := policyGlobals(source)
	defer cleanup()
	if err != nil {
		return d, err
	}
	args := make(starlark.Tuple, 0, 3)
	for _, v := range []any{context, state, outcome} {
		raw, err := json.Marshal(v)
		if err != nil || len(raw) > MaxJSON {
			return d, errors.New("policy input exceeds JSON limit")
		}
		var plain any
		if err = json.Unmarshal(raw, &plain); err != nil {
			return d, err
		}
		value, err := toStar(plain, 0)
		if err != nil {
			return d, err
		}
		value.Freeze()
		args = append(args, value)
	}
	result, err := starlark.Call(t, g["step"], args, nil)
	if err != nil {
		return d, err
	}
	budget := MaxJSON
	value, err := fromStar(result, 0, &budget)
	if err != nil {
		return d, err
	}
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > MaxJSON {
		return d, errors.New("policy output exceeds JSON limit")
	}
	decErr := decode(raw, &d)
	if decErr != nil {
		return d, decErr
	}
	if len(d.Reason) > 512 {
		return d, errors.New("policy reason too long")
	}
	switch d.Action {
	case "run", "wait", "done", "needs_plan":
	default:
		return d, errors.New("unknown policy action")
	}
	if d.Action == "wait" && (!finite(d.WaitS) || d.WaitS < 1 || d.WaitS > 30) {
		return d, errors.New("policy wait must be 1..30 seconds")
	}
	if len(d.State) == 0 {
		d.State = state
	}
	return d, nil
}
func toStar(v any, depth int) (starlark.Value, error) {
	if depth > 24 {
		return nil, errors.New("JSON nesting limit")
	}
	switch x := v.(type) {
	case nil:
		return starlark.None, nil
	case bool:
		return starlark.Bool(x), nil
	case string:
		return starlark.String(x), nil
	case float64:
		return starlark.Float(x), nil
	case []any:
		out := make([]starlark.Value, 0, len(x))
		for _, item := range x {
			v, e := toStar(item, depth+1)
			if e != nil {
				return nil, e
			}
			out = append(out, v)
		}
		return starlark.NewList(out), nil
	case map[string]any:
		out := starlark.NewDict(len(x))
		for k, item := range x {
			v, e := toStar(item, depth+1)
			if e != nil {
				return nil, e
			}
			if e = out.SetKey(starlark.String(k), v); e != nil {
				return nil, e
			}
		}
		return out, nil
	}
	return nil, errors.New("unsupported JSON value")
}
func fromStar(v starlark.Value, depth int, budget *int) (any, error) {
	*budget--
	if depth > 24 || *budget < 0 {
		return nil, errors.New("policy output nesting/node limit")
	}
	switch x := v.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.Bool:
		return bool(x), nil
	case starlark.String:
		*budget -= len(x)
		if len(x) > MaxJSON || *budget < 0 {
			return nil, errors.New("policy string limit")
		}
		return string(x), nil
	case starlark.Float:
		if !finite(float64(x)) {
			return nil, errors.New("nonfinite value")
		}
		return float64(x), nil
	case starlark.Int:
		n, ok := x.Int64()
		if !ok {
			return nil, errors.New("integer limit")
		}
		return n, nil
	case *starlark.List:
		out := make([]any, 0, x.Len())
		for i := 0; i < x.Len(); i++ {
			v, e := fromStar(x.Index(i), depth+1, budget)
			if e != nil {
				return nil, e
			}
			out = append(out, v)
		}
		return out, nil
	case *starlark.Dict:
		out := map[string]any{}
		for _, pair := range x.Items() {
			k, ok := starlark.AsString(pair[0])
			*budget -= len(k) + 3
			if !ok || *budget < 0 {
				return nil, errors.New("JSON keys must be strings")
			}
			v, e := fromStar(pair[1], depth+1, budget)
			if e != nil {
				return nil, e
			}
			out[k] = v
		}
		return out, nil
	}
	return nil, errors.New("policy must return JSON values")
}

func scalarLiteral(e syntax.Expr) bool {
	switch x := e.(type) {
	case *syntax.Literal:
		return x.Token == syntax.STRING || x.Token == syntax.INT || x.Token == syntax.FLOAT
	case *syntax.Ident:
		return x.Name == "True" || x.Name == "False" || x.Name == "None"
	case *syntax.UnaryExpr:
		_, ok := x.X.(*syntax.Literal)
		return x.Op == syntax.MINUS && ok
	}
	return false
}
