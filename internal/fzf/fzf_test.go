package fzf

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestFindEqualsFzfForJS(t *testing.T) {
	data, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Texts   []string
		Queries []struct {
			Query   string
			Results [][4]int
		}
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	for _, q := range golden.Queries {
		got := [][4]int{}
		for _, r := range Find(golden.Texts, q.Query) {
			got = append(got, [4]int{r.Index, r.Start, r.End, r.Score})
		}
		if !reflect.DeepEqual(got, q.Results) {
			t.Errorf("%q:\n got %v\nwant %v", q.Query, got, q.Results)
		}
	}
}
