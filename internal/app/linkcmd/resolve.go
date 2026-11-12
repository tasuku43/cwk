// Package linkcmd interprets locally supplied message links without I/O.
package linkcmd

import (
	"context"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// Resolve returns syntactic references without checking a remote target.
func Resolve(ctx context.Context, link string) (chatwork.MessageLinkReferences, error) {
	if err := ctx.Err(); err != nil {
		return chatwork.MessageLinkReferences{}, err
	}
	return chatwork.ParseMessageLink(link)
}
