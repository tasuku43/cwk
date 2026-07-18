// Package chatworkcmd owns application use cases for task-oriented Chatwork
// operations. It depends on a semantic task port, not on HTTP or wire DTOs.
package chatworkcmd

import (
	"context"

	"github.com/tasuku43/cwk/internal/app/portcheck"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

// Port executes one validated semantic task using the exact infrastructure
// authentication binding admitted by the application authentication gate.
type Port interface {
	Execute(context.Context, authn.BindingID, chatwork.Request) (chatwork.Result, error)
}

type Service struct {
	port Port
}

func New(port Port) *Service {
	return &Service{port: port}
}

// Execute validates the task and binding before crossing the external port.
// It returns no result when cancellation or a contract failure is observed.
func (s *Service) Execute(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	if ctx == nil {
		return chatwork.Result{}, fault.New(fault.KindContract, "missing_context", "Chatwork task context is not configured", false)
	}
	if err := ctx.Err(); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindCanceled, "operation_canceled", "Chatwork task was canceled before execution", true, err)
	}
	if err := binding.Validate(); err != nil {
		return chatwork.Result{}, fault.New(fault.KindAuthentication, "invalid_authentication_binding", "Chatwork authentication binding is invalid", false)
	}
	if err := request.Validate(); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindInvalidInput, "invalid_chatwork_task", "Chatwork task input is invalid", false, err)
	}
	if s == nil || portcheck.IsNil(s.port) {
		return chatwork.Result{}, fault.New(fault.KindContract, "missing_chatwork_port", "Chatwork task adapter is not configured", false)
	}
	result, err := s.port.Execute(ctx, binding, request)
	if err != nil {
		if structured, ok := fault.PublicCopy(err); ok {
			return chatwork.Result{}, structured
		}
		if ctx.Err() != nil {
			return chatwork.Result{}, fault.Wrap(fault.KindCanceled, "operation_canceled", "Chatwork task was canceled during execution", true, ctx.Err())
		}
		return chatwork.Result{}, fault.New(fault.KindInternal, "unclassified_chatwork_error", "Chatwork task returned an unclassified error", false)
	}
	if err := ctx.Err(); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindCanceled, "operation_canceled", "Chatwork task was canceled after execution", true, err)
	}
	if result.Task != request.Task {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_result_mismatch", "Chatwork task adapter returned a result for a different task", false)
	}
	return result, nil
}
