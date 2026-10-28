package repo

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
)

type namedRepo struct {
	proto.Repository
	name string
}

func (r namedRepo) Name() string { return r.name }

// An empty repository has no readme to load, so the readme tab must stop
// showing the loading spinner once the empty-repo message arrives.
func TestReadmeEmptyRepoStopsLoading(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := config.WithContext(context.Background(), cfg)
	r := NewReadme(common.NewCommon(ctx, 80, 24))
	r.SetSize(80, 24)

	if !r.isLoading {
		t.Fatal("new readme should start loading")
	}

	r.Update(RepoMsg(namedRepo{name: "empty"}))
	r.Update(EmptyRepoMsg{})

	if r.isLoading {
		t.Fatal("isLoading = true after EmptyRepoMsg, want false")
	}
	if strings.Contains(r.View(), "loading") {
		t.Errorf("View() still shows the spinner:\n%s", r.View())
	}
}
