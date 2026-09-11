package steam

import (
	"encoding/json"
	"testing"
)

func TestBuildMetadata(t *testing.T) {
	plan := &SyncPlan{AppID: 3445480, Branch: "alphatest", BuildID: 23992167}
	c := &ChannelPlan{Name: "linux", Depots: []*DepotPlan{{ID: 3445483, GID: 505195365870867014}}}

	raw, err := json.Marshal(buildMetadata(plan, c))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"steam":{"app_id":3445480,"branch":"alphatest","build_id":23992167,"depots":[{"gid":"505195365870867014","id":3445483}]}}`
	if string(raw) != want {
		t.Fatalf("got %s", raw)
	}
}
