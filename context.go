package httprouter

import "context"

// ContextParams returns the params map associated with the given context if one exists. Otherwise, an empty map is returned.
const ParamsContextKey = "params.context.key"

// ParamsContextKey is used to retrieve a path's params map from a request's context.
func ContextParams(ctx context.Context) Params {
	if p, ok := ctx.Value(ParamsContextKey).(Params); ok {
		return p
	}
	return Params{}
}
