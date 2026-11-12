package analysis

import (
	"strings"
	"testing"
)

// jsonlog and csvlog keep the plan's lines: a multi-line Query Text and the
// Query Parameters line must not end up in the plan.
func TestPlanTextSkipsMultiLineQueryText(t *testing.T) {
	msg := "duration: 2118.349 ms  plan:\nQuery Text: PREPARE s AS SELECT *\n  FROM t\n  WHERE a = $1;\nQuery Parameters: $1 = '1'\nSeq Scan on t  (cost=0.00..1.00 rows=1 width=4) (actual time=0.01..0.02 rows=1 loops=1)\n  Filter: (a = 1)"
	got := extractPlanText(msg)
	if !strings.HasPrefix(got, "Seq Scan on t") || strings.Contains(got, "Query Parameters") || strings.Contains(got, "FROM t") {
		t.Errorf("plan = %q", got)
	}
}

// stderr flattens the plan onto one line: a parallel plan must keep its top
// Gather Merge node instead of starting at the Sort below it.
func TestFlatPlanKeepsTopGather(t *testing.T) {
	msg := "duration: 2118.349 ms  plan: Query Text: SELECT * FROM t ORDER BY a Gather Merge  (cost=1.00..2.00 rows=10 width=8) (actual time=1.0..2.0 rows=10 loops=1) Workers Planned: 2 Workers Launched: 2 ->  Sort  (cost=1.00..1.50 rows=5 width=8) (actual time=1.0..1.5 rows=5 loops=3) Sort Key: a"
	if got := extractPlanText(msg); !strings.HasPrefix(got, "Gather Merge") {
		t.Errorf("plan = %q", got)
	}
}
