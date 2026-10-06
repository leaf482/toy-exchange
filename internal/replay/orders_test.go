package replay

import (
	"testing"

	"github.com/leaf482/toy-exchange/internal/engine"
)

func TestEngineOpsCopiesLimitsAndCancels(t *testing.T) {
	ops := EngineOps([]Action{
		{Kind: ActionLimit, ID: 1, Side: SideBuy, Price: 100, Qty: 5, Time: 10},
		{Kind: ActionCancel, CancelID: 1, Time: 12},
		{Kind: ActionLimit, ID: 2, Side: SideSell, Price: 110, Qty: 4, Time: 12},
	})
	if len(ops) != 3 {
		t.Fatalf("len %d", len(ops))
	}
	if ops[0].Kind != engine.OpLimit || ops[0].In.ID != 1 || ops[0].In.Side != engine.SideBuy || ops[0].In.Type != engine.TypeLimit || ops[0].In.Price != 100 || ops[0].In.Qty != 5 || ops[0].In.Timestamp != 10 {
		t.Fatalf("limit %+v", ops[0])
	}
	if ops[1].Kind != engine.OpCancel || ops[1].CancelID != 1 || ops[1].In.Timestamp != 12 {
		t.Fatalf("cancel %+v", ops[1])
	}
	if ops[2].Kind != engine.OpLimit || ops[2].In.Side != engine.SideSell || ops[2].In.ID != 2 || ops[2].In.Qty != 4 {
		t.Fatalf("sell %+v", ops[2])
	}
}

func TestEngineOpsEmpty(t *testing.T) {
	if EngineOps(nil) != nil {
		t.Fatal("nil actions")
	}
	if EngineOps([]Action{}) != nil {
		t.Fatal("empty actions")
	}
}
