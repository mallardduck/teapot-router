# Migrating from v0.15.2 to the next release

This lists the changes since the last tag, `v0.15.2`, that affect code using teapot-router. Each entry says what
breaks and how to update.

## Breaking changes

### `Resource` takes a controller and options

`ResourceHandlers`, `ResourceHandlerBuilder`, and `NewResourceHandlers` are removed.

Before:

```go
r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{
    Index: http.HandlerFunc(listPhotos),
    Show:  http.HandlerFunc(showPhoto),
    StoreMethod: teapot.PUT,
    UpdateMethod: teapot.POST,
})
```

After:

```go
type PhotoController struct{}

func (PhotoController) Index(w http.ResponseWriter, r *http.Request) { listPhotos(w, r) }
func (PhotoController) Show(w http.ResponseWriter, r *http.Request)  { showPhoto(w, r) }

r.Resource("photos", PhotoController{}, teapot.StoreVerb(teapot.PUT))
```

- The path defaults to `/` plus the name. Use `teapot.Path(...)` for another path.
- The parameter defaults to the singular of the name. Use `teapot.Param(...)` for another name.
- `UpdateMethod` is removed. Update now registers both `PUT` and `PATCH`.
- `Head` is removed. See the HEAD change below.
- `StoreVerb` must be `POST` or `PUT`. Other values panic at registration.

### `Router.Func()` is removed

`FuncRouter` and its methods are removed. Replace each call with `http.HandlerFunc`:

```go
// before
r.Func().GET("/users", listUsers)

// after
r.GET("/users", http.HandlerFunc(listUsers))
```

### `dispatch` handlers take `http.Handler`

`dispatch.Builder.Default` and `dispatch.Route.Do` now take `http.Handler`, matching `teapot.Dispatch`. For a function
value, use the new `DefaultFunc` and `DoFunc`:

```go
// before
b.Default(listV1)
b.When(dispatch.QueryExists("acl")).Do(getACL)

// after
b.DefaultFunc(listV1)
b.When(dispatch.QueryExists("acl")).DoFunc(getACL)
```

A router or any other `http.Handler` can now be passed directly, for example `Do(vhostRouter)`.

## Behavior changes

- **HEAD fallback.** A HEAD request with no HEAD route is served by the GET route for the same path. Before, it returned
  405. An explicit HEAD route still wins.
- **Host matching.** A trailing dot on a host (`bucket.example.com.`) is ignored, and matching is case-insensitive.
  Generated URLs are lowercase and never carry a trailing dot.
- **`SubdomainURL`** adds a leading slash when the path lacks one.

## Additions

- `Router.RegisterDebugRoute(path, name)`: registers the route listing endpoint.
- `Matchers.HostHasSubdomain(domain)` for `Dispatch` callbacks.
- `DefaultFunc` and `DoFunc` on `dispatch`.
- The HTML route table is sortable by method, pattern, and name.
- `Only`, `Except`, `Middleware`, `Path`, `Param`, `Names`, and `StoreVerb` options for `Resource`.

## Dependencies

- New: `github.com/jinzhu/inflection`, used for the default `Resource` parameter name.
