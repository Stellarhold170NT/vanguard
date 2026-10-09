// qa-label applies the w4-04 DRY-RUN provisional verdicts to the machine
// sheet — PIPELINE QA ONLY (protocol §0: the gate labels the real sheet;
// this artifact is evidence that the merge path works end-to-end and is
// never presented as the audit result).
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// dry-run verdicts: id -> (verdict, reason_code, note)
var labels = map[string][3]string{
	"F-009": {"FP", "heuristic-context", "login is an action endpoint, not a resource create — the create-shaped heuristic over-fires on auth verbs (protocol §4 dispute example)"},
	"F-018": {"FP", "heuristic-context", "byte[] is a file download, not an entity collection — pagination-metadata reasoning does not apply"},
	"F-022": {"FP", "heuristic-context", "byte[] file download — same as F-018"},
	"F-023": {"FP", "heuristic-context", "/api is an infrastructural version prefix and a route index is an aggregate, not an unpaginated entity collection"},
}

func main() {
	in, out := os.Args[1], os.Args[2]
	f, err := os.Open(in)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		panic(err)
	}
	header := rows[0]
	id, v, r, n := col(header, "id"), col(header, "gate_verdict"), col(header, "reason_code"), col(header, "note")
	fs := col(header, "from_slots")
	tp, fp := 0, 0
	for _, row := range rows[1:] {
		l, ok := labels[row[id]]
		if ok {
			row[v], row[r], row[n] = l[0], l[1], "qa-dry-run: "+l[2]
			fp++
		} else {
			row[v] = "TP"
			if strings.TrimSpace(row[fs]) == "" {
				row[n] = "qa-dry-run: unclaimed firing, verified against the charter predicate"
			} else {
				row[n] = "qa-dry-run: construct matches the pre-registered pattern"
			}
			tp++
		}
	}
	wf, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer wf.Close()
	w := csv.NewWriter(wf)
	if err := w.WriteAll(rows); err != nil {
		panic(err)
	}
	w.Flush()
	fmt.Printf("qa-label: %d rows -> %d TP, %d FP (dry-run)\n", len(rows)-1, tp, fp)
}

func col(header []string, name string) int {
	for i, h := range header {
		if strings.TrimSpace(h) == name {
			return i
		}
	}
	panic("no column " + name)
}
