package client

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestClientDoesNotLinkTheServer guards the client's dependency footprint: a
// program that only talks to a Colonies server must not link the server's own
// machinery (the embedded etcd cluster, the gin web framework). Importing
// pkg/cluster for its configuration type used to pull all of it in.
func TestClientDoesNotLinkTheServer(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go tool not on PATH")
	}
	cmd := exec.Command("go", "list", "-deps", ".")
	// The module may sit inside a parent folder with its own go.work; list it on its own.
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	// pkg/cluster itself embeds etcd; its subpackage pkg/cluster/config is plain types and allowed.
	forbiddenPackages := []string{"github.com/colonyos/colonies/pkg/cluster"}
	forbiddenTrees := []string{"go.etcd.io/etcd/server/", "github.com/gin-gonic/gin"}
	for _, dep := range strings.Fields(string(out)) {
		for _, f := range forbiddenPackages {
			if dep == f {
				t.Errorf("pkg/client depends on %s", dep)
			}
		}
		for _, f := range forbiddenTrees {
			if dep == strings.TrimSuffix(f, "/") || strings.HasPrefix(dep, strings.TrimSuffix(f, "/")+"/") {
				t.Errorf("pkg/client depends on %s", dep)
			}
		}
	}
}
