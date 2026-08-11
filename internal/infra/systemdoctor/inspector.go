// Package systemdoctor provides the local, offline doctor adapter.
package systemdoctor

import (
	"context"
	"fmt"
	"runtime"

	"github.com/tasuku43/cwk/internal/domain/doctor"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
)

// Inspector reports properties of the running binary without network access.
type Inspector struct {
	inspectToken func() chatworkapi.TokenEnvironmentInspection
}

// New creates the default system inspector.
func New() *Inspector {
	return &Inspector{inspectToken: chatworkapi.InspectTokenEnvironment}
}

// Inspect implements doctorcmd.InspectorPort by structural typing.
func (i *Inspector) Inspect(ctx context.Context) (doctor.Report, error) {
	if ctx == nil {
		return doctor.Report{}, fmt.Errorf("inspection context is nil")
	}
	if err := ctx.Err(); err != nil {
		return doctor.Report{}, err
	}
	inspectToken := chatworkapi.InspectTokenEnvironment
	if i != nil && i.inspectToken != nil {
		inspectToken = i.inspectToken
	}
	token := inspectToken()
	tokenStatus := doctor.CheckStatusFail
	if token.State == chatworkapi.TokenEnvironmentValid {
		tokenStatus = doctor.CheckStatusPass
	}
	return doctor.Report{Checks: []doctor.Check{
		{
			Name:   "runtime",
			Status: doctor.CheckStatusPass,
			Detail: fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		},
		{
			Name:   "chatwork-token",
			Status: tokenStatus,
			Detail: fmt.Sprintf("state=%s source=%s name=%s config=%s keychain=%s", token.State, token.Source, token.Name, token.Config, token.Keychain),
		},
	}}, nil
}
