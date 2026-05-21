package template

import (
	"context"
	"net/http"
)

type identityContextKey struct{}

func AttachIdentity(r *http.Request, identity *Identity) *http.Request {
	return r.WithContext(withIdentity(r.Context(), identity))
}

func IdentityFromRequest(r *http.Request) (*Identity, bool) {
	value := r.Context().Value(identityContextKey{})
	identity, ok := value.(*Identity)
	return identity, ok && identity != nil
}

func withIdentity(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}
