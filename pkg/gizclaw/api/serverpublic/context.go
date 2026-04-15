package serverpublic

import "context"

type callerPublicKeyContextKey string

const callerPublicKeyKey callerPublicKeyContextKey = "caller_public_key"

func WithCallerPublicKey(ctx context.Context, publicKey string) context.Context {
	return context.WithValue(ctx, callerPublicKeyKey, publicKey)
}

func CallerPublicKey(ctx context.Context) string {
	value, _ := ctx.Value(callerPublicKeyKey).(string)
	return value
}
