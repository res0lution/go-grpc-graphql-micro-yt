package template

import (
	"context"
	"net/http"
)

type identityContextKey struct{}
type userInfoContextKey struct{}

func AttachIdentity(r *http.Request, identity *Identity) *http.Request {
	return r.WithContext(withIdentity(r.Context(), identity))
}

func IdentityFromRequest(r *http.Request) (*Identity, bool) {
	value := r.Context().Value(identityContextKey{})
	identity, ok := value.(*Identity)
	return identity, ok && identity != nil
}

func AttachUserInfo(r *http.Request, userInfo *UserInfo) *http.Request {
	return r.WithContext(withUserInfo(r.Context(), userInfo))
}

func UserInfoFromRequest(r *http.Request) (*UserInfo, bool) {
	value := r.Context().Value(userInfoContextKey{})
	userInfo, ok := value.(*UserInfo)
	return userInfo, ok && userInfo != nil
}

func withIdentity(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}

func withUserInfo(ctx context.Context, userInfo *UserInfo) context.Context {
	return context.WithValue(ctx, userInfoContextKey{}, userInfo)
}
