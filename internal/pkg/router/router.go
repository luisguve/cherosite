package router

import(
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/luisguve/cheropatilla/internal/pkg/livedata"
	pb "github.com/luisguve/cheropatilla/internal/pkg/cheropatillapb"
)

type Router struct {
	handler    *mux.Router
	upgrader   websocket.Upgrader
	crudClient *pb.CrudCheropatillaClient
	templates  template.Template
	store      sessions.Store
	hub        *livedata.Hub
}

func New(t *template.Template, cc *pb.CrudCheropatillaClient, s sessions.Store, 
	hub *livedata.Hub) *Router {
	if t == nil {
		log.Fatal("missing templates")
	}
	if cc == nil {
		log.Fatal("missing crud client")
	}
	if s == nil {
		log.Fatal("missing sessions store")
	}
	if hub == nil {
		log.Fatal("missing hub")
	}
	return &Router{
		handler:    mux.NewRouter(),
		upgrader:   websocket.Upgrader{
			ReadBufferSize:  livedata.ReadBufferSize,
			WriteBufferSize: livedata.WriteBufferSize,
		},
		crudClient: cc,
		templates:  t,
		store:      s,
		hub:        hub,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.handler.ServeHTTP(w, r)
}

func (r *Router) SetupRoutes() {
	root := r.handler.PathPrefix("/").Subrouter()
	// favicon (not found)
	root.Handle("/favicon", http.NotFoundHandler)
	//
	// WEBSOCKET
	//
	root.HandleFunc("/livenotifs", r.handleLiveNotifs).Methods("GET")
	.Headers("X-Requested-With", "XMLHttpRequest")
	//
	// GET DASHBOARD OR LOGIN PAGE
	// matches GET "/"
	root.HandleFunc("/", r.onlyUsers(userContentsHandler(r.handleRoot))).Methods("GET")
	//
	// GET RECYCLE USER FEED
	// matches GET "/recycle"
	root.HandleFunc("/recycle", r.onlyUsers(userContentsHandler(r.handleRecycleFeed)))
	.Methods("GET")
	//
	// GET EXPLORE PAGE
	// matches GET "/explore"
	root.HandleFunc("/explore", r.handleExplore).Methods("GET")
	//
	// GET RECYCLE EXPLORE FEED
	// matches GET "/explore/recycle"
	root.HandleFunc("/explore/recycle", 
		r.onlyUsers(userContentsHandler(r.handleExploreRecycle))).Methods("GET")
		.headers("X-Requested-With", "XMLHttpRequest")
	//
	// GET REQUEST TO READ ALL NOTIFICATIONS FROM THIS USER
	// matches GET "/readall"
	root.HandleFunc("/readall", r.handleReadAll).Methods("GET")
		.Headers("X-Requested-With", "XMLHttpRequest")
	//
	// SECTION LEVEL HANDLERS
	//
	section := root.PathPrefix("/{section}").Subrouter()
	//
	// GET SECTION THREADS
	// matches GET "/{section}"
	section.HandleFunc("/", r.handleViewSection).Methods("GET")
	//
	// POST A THREAD IN A SECTION
	// matches POST "/{section}/new"
	section.HandleFunc("/new", r.onlyUsers(userContentsHandler(r.handleNewThread)))
	.Methods("POST")
	//
	// GET RECYCLE SECTION FEED
	// matches GET "/{section}/recycle"
	section.HandleFunc("/recycle", r.handleRecycleSection).Methods("GET")
	//
	// THREAD LEVEL HANDLERS
	//
	thread := section.PathPrefix("/{thread}").Subrouter()
	//
	// GET A THREAD IN A SECTION AND ITS COMMENTS
	// matches GET "/{section}/{thread}/"
	thread.HandleFunc("/", r.handleViewThread).Methods("GET")
	//
	// GET RECYCLE COMMENTS
	// matches GET "/{section}/{thread}/recycle"
	thread.HandleFunc("/recycle", r.handleRecycleComments).Methods("GET")
	//
	// COMMENT LEVEL HANDLERS
	//
	comments := thread.PathPrefix("/comment").Subrouter()
	//
	// GET SUBCOMMENTS OF A COMMENT IN JSON FORMAT
	// matches GET "/{section}/{thread}/comment/?c_id={c_id}&offset={offset}"
	comments.HandleFunc("/", r.handleGetSubcomments).Methods("GET")
	.Headers("X-Requested-With", "XMLHttpRequest").Queries("c_id", "{c_id:[a-zA-Z0-9]+}",
	 "offset", "{offset:[0-9]+}")
	//
	// POST A COMMENT IN A THREAD
	// matches POST "/{section}/{thread}/comment/"
	comments.HandleFunc("/", r.onlyUsers(userContentsHandler(r.handlePostComment)))
	.Methods("POST")
	//
	// POST A SUBCOMMENT
	// matches POST "/{section}/{thread}/comment/?c_id={c_id}"
	comments.HandleFunc("/", r.onlyUsers(userContentsHandler(r.handlePostSubcomment)))
	.Methods("POST").Queries("c_id", "{c_id:[a-zA-Z0-9]+}")
	// UPVOTES
	upvotes := thread.PathPrefix("/upvote").Subrouter()
	//
	// POST AN UPVOTE TO A THREAD
	// matches POST "/{section}/{thread}/upvote/"
	upvotes.HandleFunc("/", r.onlyUsers(userContentsHandler(r.handleUpvoteThread)))
	.Methods("POST")
	//
	// POST AN UPVOTE TO A COMMENT
	// matches POST "/{section}/{thread}/upvote/?c_id={c_id}"
	upvotes.HandleFunc("/", r.onlyUsers(userContentsHandler(r.handleUpvoteComment)))
	.Methods("POST").Queries("c_id", "{c_id:[a-zA-Z0-9]+}")
	//
	// POST AN UPVOTE TO A SUBCOMMENT
	// matches POST "/{section}/{thread}/upvote/?c_id={c_id}&sc_id={sc_id}"
	upvotes.HandleFunc("/", 
		r.onlyUsers(userContentsHandler(r.handleUpvoteSubcomment))).Methods("POST")
		.Queries("c_id", "{c_id:[a-zA-Z0-9]+}", "sc_id", "{sc_id:[a-zA-Z0-9]+}")
}