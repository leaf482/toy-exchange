package replay

import "github.com/leaf482/toy-exchange/internal/engine"

// EngineOps copies finished synthetic actions into the engine script.
// A limit becomes an OrderInput. A cancel keeps the id and timestamp.
// The matching engine is not called here. The caller runs the slice after
// synthesis has finished.
func EngineOps(actions []Action) []engine.Op {
	if len(actions) == 0 {
		return nil
	}
	out := make([]engine.Op, len(actions))
	for i, a := range actions {
		if a.Kind == ActionCancel {
			out[i] = engine.Op{
				Kind:     engine.OpCancel,
				CancelID: a.CancelID,
				In:       engine.OrderInput{Timestamp: a.Time},
			}
			continue
		}
		side := engine.SideBuy
		if a.Side == SideSell {
			side = engine.SideSell
		}
		out[i] = engine.Op{
			Kind: engine.OpLimit,
			In: engine.OrderInput{
				ID:        a.ID,
				Side:      side,
				Type:      engine.TypeLimit,
				Price:     a.Price,
				Qty:       a.Qty,
				Timestamp: a.Time,
			},
		}
	}
	return out
}
