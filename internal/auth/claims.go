package auth

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/metadata"
)

type contextKey int

const userIDKey contextKey = iota

var ErrUnauthenticated = errors.New("unauthenticated")

func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromContext(ctx context.Context) (int64, error) {
	v, ok := ctx.Value(userIDKey).(int64)
	if !ok || v == 0 {
		return 0, ErrUnauthenticated
	}
	return v, nil
}

// ExtractBearerToken pulls the JWT from the gRPC incoming metadata.
func ExtractBearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrUnauthenticated
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return "", ErrUnauthenticated
	}
	parts := strings.SplitN(vals[0], " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", ErrUnauthenticated
	}
	return parts[1], nil
}
