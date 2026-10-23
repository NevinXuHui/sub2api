package service

import (
	"context"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type accountListGroupsKey struct{}

// ParseAccountListGroups retains the single-group API while carrying an optional
// union filter through paginated listing, exports and bulk operations.
func ParseAccountListGroups(ctx context.Context, value string) (context.Context, int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ctx, 0, nil
	}
	parts := strings.Split(value, ",")
	if len(parts) > 1000 {
		return ctx, 0, infraerrors.BadRequest("INVALID_GROUP_FILTER", "too many groups")
	}
	ids := make([]int64, 0, len(parts))
	seen := map[int64]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		id := AccountListGroupUngrouped
		if part != "ungrouped" {
			parsed, err := strconv.ParseInt(part, 10, 64)
			if err != nil || parsed < 0 || (parsed == 0 && len(parts) > 1) {
				return ctx, 0, infraerrors.BadRequest("INVALID_GROUP_FILTER", "invalid group filter")
			}
			id = parsed
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 1 {
		return ctx, ids[0], nil
	}
	return context.WithValue(ctx, accountListGroupsKey{}, ids), 0, nil
}

func AccountListGroups(ctx context.Context) []int64 {
	ids, _ := ctx.Value(accountListGroupsKey{}).([]int64)
	return ids
}
