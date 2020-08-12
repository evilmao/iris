package router

import (
	"net/http"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/errgroup"
	"github.com/kataras/iris/v12/macro"
)

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party could also be named as 'Join' or 'Node' or 'Group' , Party chosen because it is fun.
//
// Look the `APIBuilder` structure for its implementation.
type Party interface {
	// IsRoot reports whether this Party is the root Application's one.
	// It will return false on all children Parties, no exception.
	IsRoot() bool

	// ConfigureContainer accepts one or more functions that can be used
	// to configure dependency injection features of this Party
	// such as register dependency and register handlers that will automatically inject any valid dependency.
	// However, if the "builder" parameter is nil or not provided then it just returns the *APIContainer,
	// which automatically initialized on Party allocation.
	//
	// It returns the same `APIBuilder` featured with Dependency Injection.
	ConfigureContainer(builder ...func(*APIContainer)) *APIContainer

	// GetRelPath returns the current party's relative path.
	// i.e:
	// if r := app.Party("/users"), then the `r.GetRelPath()` is the "/users".
	// if r := app.Party("www.") or app.Subdomain("www") then the `r.GetRelPath()` is the "www.".
	GetRelPath() string
	// GetReporter returns the reporter for adding or receiving any errors caused when building the API.
	GetReporter() *errgroup.Group
	// Macros returns the macro collection that is responsible
	// to register custom macros with their own parameter types and their macro functions for all routes.
	//
	// Learn more at:  https://github.com/kataras/iris/tree/master/_examples/routing/dynamic-path
	Macros() *macro.Macros

	// OnErrorCode registers a handlers chain for this `Party` for a specific HTTP status code.
	// Read more at: http://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
	// Look `OnAnyErrorCode` too.
	OnErrorCode(statusCode int, handlers ...context.Handler) []*Route
	// OnAnyErrorCode registers a handlers chain for all error codes
	// (4xxx and 5xxx, change the `ClientErrorCodes` and `ServerErrorCodes` variables to modify those)
	// Look `OnErrorCode` too.
	OnAnyErrorCode(handlers ...context.Handler) []*Route

	// Party returns a new child Party which inherites its
	// parent's options and middlewares.
	// If "relativePath" matches the parent's one then it returns the current Party.
	// A Party groups routes which may have the same prefix or subdomain and share same middlewares.
	//
	// To create a group of routes for subdomains
	// use the `Subdomain` or `WildcardSubdomain` methods
	// or pass a "relativePath" as "admin." or "*." respectfully.
	Party(relativePath string, middleware ...context.Handler) Party
	// PartyFunc same as `Party`, groups routes that share a base path or/and same handlers.
	// However this function accepts a function that receives this created Party instead.
	// Returns the Party in order the caller to be able to use this created Party to continue the
	// top-bottom routes "tree".
	//
	// Note: `iris#Party` and `core/router#Party` describes the exactly same interface.
	//
	// Usage:
	// app.PartyFunc("/users", func(u iris.Party){
	//	u.Use(authMiddleware, logMiddleware)
	//	u.Get("/", getAllUsers)
	//	u.Post("/", createOrUpdateUser)
	//	u.Delete("/", deleteUser)
	// })
	//
	// Look `Party` for more.
	PartyFunc(relativePath string, partyBuilderFunc func(p Party)) Party
	// Subdomain returns a new party which is responsible to register routes to
	// this specific "subdomain".
	//
	// If called from a child party then the subdomain will be prepended to the path instead of appended.
	// So if app.Subdomain("admin").Subdomain("panel") then the result is: "panel.admin.".
	Subdomain(subdomain string, middleware ...context.Handler) Party

	// UseRouter upserts one or more handlers that will be fired
	// right before the main router's request handler.
	//
	// Use this method to register handlers, that can ran
	// independently of the incoming request's values,
	// that they will be executed ALWAYS against ALL children incoming requests.
	// Example of use-case: CORS.
	//
	// Note that because these are executed before the router itself
	// the Context should not have access to the `GetCurrentRoute`
	// as it is not decided yet which route is responsible to handle the incoming request.
	// It's one level higher than the `WrapRouter`.
	// The context SHOULD call its `Next` method in order to proceed to
	// the next handler in the chain or the main request handler one.
	UseRouter(handlers ...context.Handler)
	// Use appends Handler(s) to the current Party's routes and child routes.
	// If the current Party is the root, then it registers the middleware to all child Parties' routes too.
	Use(middleware ...context.Handler)
	// UseOnce either inserts a middleware,
	// or on the basis of the middleware already existing,
	// replace that existing middleware instead.
	UseOnce(handlers ...context.Handler)
	// Done appends to the very end, Handler(s) to the current Party's routes and child routes.
	// The difference from .Use is that this/or these Handler(s) are being always running last.
	Done(handlers ...context.Handler)

	// Reset removes all the begin and done handlers that may derived from the parent party via `Use` & `Done`,
	// and the execution rules.
	// Note that the `Reset` will not reset the handlers that are registered via `UseGlobal` & `DoneGlobal`.
	//
	// Returns this Party.
	Reset() Party

	// AllowMethods will re-register the future routes that will be registered
	// via `Handle`, `Get`, `Post`, ... to the given "methods" on that Party and its children "Parties",
	// duplicates are not registered.
	//
	// Call of `AllowMethod` will override any previous allow methods.
	AllowMethods(methods ...string) Party

	// SetExecutionRules alters the execution flow of the route handlers outside of the handlers themselves.
	//
	// For example, if for some reason the desired result is the (done or all) handlers to be executed no matter what
	// even if no `ctx.Next()` is called in the previous handlers, including the begin(`Use`),
	// the main(`Handle`) and the done(`Done`) handlers themselves, then:
	// Party#SetExecutionRules(iris.ExecutionRules {
	//   Begin: iris.ExecutionOptions{Force: true},
	//   Main:  iris.ExecutionOptions{Force: true},
	//   Done:  iris.ExecutionOptions{Force: true},
	// })
	//
	// Note that if : true then the only remained way to "break" the handler chain is by `ctx.StopExecution()` now that `ctx.Next()` does not matter.
	//
	// These rules are per-party, so if a `Party` creates a child one then the same rules will be applied to that as well.
	// Reset of these rules (before `Party#Handle`) can be done with `Party#SetExecutionRules(iris.ExecutionRules{})`.
	//
	// The most common scenario for its use can be found inside Iris MVC Applications;
	// when we want the `Done` handlers of that specific mvc app's `Party`
	// to be executed but we don't want to add `ctx.Next()` on the `OurController#EndRequest`.
	//
	// Returns this Party.
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/mvc/middleware/without-ctx-next
	SetExecutionRules(executionRules ExecutionRules) Party
	// SetRegisterRule sets a `RouteRegisterRule` for this Party and its children.
	// Available values are:
	// * RouteOverride (the default one)
	// * RouteSkip
	// * RouteError
	// * RouteOverlap.
	SetRegisterRule(rule RouteRegisterRule) Party

	// Handle registers a route to the server's router.
	// if empty method is passed then handler(s) are being registered to all methods, same as .Any.
	//
	// Returns the read-only route information.
	Handle(method string, registeredPath string, handlers ...context.Handler) *Route
	// HandleMany works like `Handle` but can receive more than one
	// paths separated by spaces and returns always a slice of *Route instead of a single instance of Route.
	//
	// It's useful only if the same handler can handle more than one request paths,
	// otherwise use `Party` which can handle many paths with different handlers and middlewares.
	//
	// Usage:
	// 	app.HandleMany(iris.MethodGet, "/user /user/{id:uint64} /user/me", userHandler)
	// At the other side, with `Handle` we've had to write:
	// 	app.Handle(iris.MethodGet, "/user", userHandler)
	// 	app.Handle(iris.MethodGet, "/user/{id:uint64}", userHandler)
	// 	app.Handle(iris.MethodGet, "/user/me", userHandler)
	//
	// This method is used behind the scenes at the `Controller` function
	// in order to handle more than one paths for the same controller instance.
	HandleMany(method string, relativePath string, handlers ...context.Handler) []*Route

	// HandleDir registers a handler that serves HTTP requests
	// with the contents of a file system (physical or embedded).
	//
	// first parameter  : the route path
	// second parameter : the file system needs to be served
	// third parameter  : not required, the serve directory options.
	//
	// Alternatively, to get just the handler for that look the FileServer function instead.
	//
	//     api.HandleDir("/static", iris.Dir("./assets"), iris.DirOptions{IndexName: "/index.html", Compress: true})
	//
	// Returns all the registered routes, including GET index and path patterm and HEAD.
	//
	// Examples can be found at: https://github.com/kataras/iris/tree/master/_examples/file-server
	HandleDir(requestPath string, fs http.FileSystem, opts ...DirOptions) []*Route

	// None registers an "offline" route
	// see context.ExecRoute(routeName) and
	// party.Routes().Online(handleResultregistry.*Route, "GET") and
	// Offline(handleResultregistry.*Route)
	//
	// Returns the read-only route information.
	None(path string, handlers ...context.Handler) *Route

	// Get registers a route for the Get HTTP Method.
	//
	// Returns the read-only route information.
	Get(path string, handlers ...context.Handler) *Route
	// Post registers a route for the Post HTTP Method.
	//
	// Returns the read-only route information.
	Post(path string, handlers ...context.Handler) *Route
	// Put registers a route for the Put HTTP Method.
	//
	// Returns the read-only route information.
	Put(path string, handlers ...context.Handler) *Route
	// Delete registers a route for the Delete HTTP Method.
	//
	// Returns the read-only route information.
	Delete(path string, handlers ...context.Handler) *Route
	// Connect registers a route for the Connect HTTP Method.
	//
	// Returns the read-only route information.
	Connect(path string, handlers ...context.Handler) *Route
	// Head registers a route for the Head HTTP Method.
	//
	// Returns the read-only route information.
	Head(path string, handlers ...context.Handler) *Route
	// Options registers a route for the Options HTTP Method.
	//
	// Returns the read-only route information.
	Options(path string, handlers ...context.Handler) *Route
	// Patch registers a route for the Patch HTTP Method.
	//
	// Returns the read-only route information.
	Patch(path string, handlers ...context.Handler) *Route
	// Trace registers a route for the Trace HTTP Method.
	//
	// Returns the read-only route information.
	Trace(path string, handlers ...context.Handler) *Route
	// Any registers a route for ALL of the HTTP methods:
	// Get
	// Post
	// Put
	// Delete
	// Head
	// Patch
	// Options
	// Connect
	// Trace
	Any(registeredPath string, handlers ...context.Handler) []*Route
	// CreateRoutes returns a list of Party-based Routes.
	// It does NOT registers the route. Use `Handle, Get...` methods instead.
	// This method can be used for third-parties Iris helpers packages and tools
	// that want a more detailed view of Party-based Routes before take the decision to register them.
	CreateRoutes(methods []string, relativePath string, handlers ...context.Handler) []*Route
	// StaticContent registers a GET and HEAD method routes to the requestPath
	// that are ready to serve raw static bytes, memory cached.
	//
	// Returns the GET *Route.
	StaticContent(requestPath string, cType string, content []byte) *Route
	// Favicon serves static favicon
	// accepts 2 parameters, second is optional
	// favPath (string), declare the system directory path of the __.ico
	// requestPath (string), it's the route's path, by default this is the "/favicon.ico" because some browsers tries to get this by default first,
	// you can declare your own path if you have more than one favicon (desktop, mobile and so on)
	//
	// this func will add a route for you which will static serve the /yuorpath/yourfile.ico to the /yourfile.ico
	// (nothing special that you can't handle by yourself).
	// Note that you have to call it on every favicon you have to serve automatically (desktop, mobile and so on).
	//
	// Returns the GET *Route.
	Favicon(favPath string, requestPath ...string) *Route

	// RegisterView registers and loads a view engine middleware for that group of routes.
	// It overrides any of the application's root registered view engines.
	// To register a view engine per handler chain see the `Context.ViewEngine` instead.
	// Read `Configuration.ViewEngineContextKey` documentation for more.
	RegisterView(viewEngine context.ViewEngine)
	// Layout overrides the parent template layout with a more specific layout for this Party.
	// It returns the current Party.
	//
	// The "tmplLayoutFile" should be a relative path to the templates dir.
	// Usage:
	//
	// app := iris.New()
	// app.RegisterView(iris.$VIEW_ENGINE("./views", ".$extension"))
	// my := app.Party("/my").Layout("layouts/mylayout.html")
	// 	my.Get("/", func(ctx iris.Context) {
	// 		ctx.View("page1.html")
	// 	})
	//
	// Examples: https://github.com/kataras/iris/tree/master/_examples/view
	Layout(tmplLayoutFile string) Party
}
