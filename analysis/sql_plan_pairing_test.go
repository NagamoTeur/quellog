package analysis

import (
	"strings"
	"testing"

	"github.com/Alain-L/quellog/parser"
)

// A statement slower than auto_explain.log_min_duration but faster than
// log_min_duration_statement logs a plan and no timed line: the plan stays
// pending on its backend. The next timed statement of that backend, an SQL
// EXECUTE whose own executor time stays under auto_explain.log_min_duration
// (most of its time goes to sending rows), used to get that plan. A statement
// that does log its own plan keeps getting it.
func TestPendingPlanOfAnotherStatementIsDropped(t *testing.T) {
	ch := make(chan []parser.LogEntry, 16)
	errc := make(chan error, 1)
	go func() { errc <- parser.ParseFile("testdata/plan_pairing.log", ch); close(ch) }()
	var entries []parser.LogEntry
	for batch := range ch {
		entries = append(entries, batch...)
	}
	if err := <-errc; err != nil {
		t.Fatal(err)
	}

	m := sqlRunSingle(entries)
	var execute, count *QueryStat
	for _, qs := range m.QueryStats {
		switch {
		case strings.HasPrefix(strings.ToLower(qs.RawQuery), "execute "):
			execute = qs
		case strings.Contains(strings.ToLower(qs.RawQuery), "count(*)"):
			count = qs
		}
	}
	if execute == nil || count == nil {
		t.Fatalf("statements not found: execute=%v count=%v", execute != nil, count != nil)
	}
	if strings.Contains(execute.LastPlan, "ORDER_LINES") {
		t.Errorf("the EXECUTE carries the ORDER_LINES plan: %q", execute.LastPlan)
	}
	if !strings.Contains(count.LastPlan, "Aggregate") {
		t.Errorf("count(*) lost its own plan: %q", count.LastPlan)
	}
}

func TestPlanBelongsTo(t *testing.T) {
	cases := []struct {
		queryText, query string
		want             bool
	}{
		{"", "select 1", true},
		{"select * from t where id = 1; seq scan on t (cost=…", "SELECT *\n  FROM t WHERE id = 1;", true},
		{"prepare qpsqlpstmt_1001 as select * from \"shipments\" gather merge …", "EXECUTE qpsqlpstmt_1001 (1, 2, 3)", true},
		{"select * from \"order_lines\" where load_ref = '00091505'; seq scan …", "EXECUTE qpsqlpstmt_1001 (1, 2, 3)", false},
		{"select * from a; seq scan on a …", "select * from b", false},
	}
	for _, c := range cases {
		if got := planBelongsTo(c.queryText, c.query); got != c.want {
			t.Errorf("planBelongsTo(%q, %q) = %v, want %v", c.queryText, c.query, got, c.want)
		}
	}
}
