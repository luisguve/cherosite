package router

import(
	"net/http"
	"html/template"
	"log"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/luisguve/cherosite/internal/pkg/livedata"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
)

type Router struct {
	handler    *mux.Router
	upgrader   websocket.Upgrader
	crudClient pbApi.CrudCheropatillaClient
	templates  *template.Template
	store      sessions.Store
	hub        *livedata.Hub
}

func New(t *template.Template, cc pbApi.CrudCheropatillaClient, s sessions.Store, 
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

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func (r *Router) SetupRoutes() {
	root := r.handler.PathPrefix("/").Subrouter()
	// favicon (not found)
	root.Handle("/favicon", http.NotFoundHandler())
	//
	// WEBSOCKET
	//
	root.HandleFunc("/livenotifs", r.handleLiveNotifs).Methods("GET").Headers("X-Requested-With", "XMLHttpRequest")
	
	// handlers for homepage "/" features
	root.HandleFunc("/", r.onlyUsers(r.handleRoot)).Methods("GET")

	root.HandleFunc("/recyclefeed", r.onlyUsers(r.handleRecycleFeed)).Methods("GET").Headers("X-Requested-With", "XMLHttpRequest")

	root.HandleFunc("/recycleactivity", r.onlyUsers(r.handleRecycleMyActivity)).Methods("GET").Headers("X-Requested-With", "XMLHttpRequest")

	root.HandleFunc("/recyclesaved", r.onlyUsers(r.handleRecycleMySaved)).Methods("GET").Headers("X-Requested-With", "XMLHttpRequest")

	// explore page
	root.HandleFunc("/explore", r.handleExplore).Methods("GET")
	root.HandleFunc("/explore/recycle", r.handleExploreRecycle).Methods("GET").Headers("X-Requested-With", "XMLHttpRequest")

	// notifications
	root.HandleFunc("/readnotifs", r.onlyUsers(r.handleReadNotifs)).Methods("GET").Headers("X-Requested-With", "XMLHttpRequest")
	root.HandleFunc("/clearnotifs", r.onlyUsers(r.handleClearNotifs)).Methods("GET").Headers("X-Requested-With", "XMLHttpRequest")

	// follow event
	root.HandleFunc("/follow", r.onlyUsers(r.handleFollow)).Methods("POST").Queries("username","{username:[a-zA-Z0-9]+}")
	// unfollow event
	root.HandleFunc("/unfollow", r.onlyUsers(r.handleUnfollow)).Methods("POST").Queries("username","{username:[a-zA-Z0-9]+}")

	// get basic info of users either following or followers
	root.HandleFunc("/viewusers", r.handleViewUsers).Methods("GET").Queries("context", "{context:[a-z]+}", "userid", "{userid:[a-zA-Z0-9]+}").Headers("X-Requested-With", "XMLHttpRequest")

	// current user's profile page
	root.HandleFunc("/myprofile", r.onlyUsers(r.handleMyProfile)).Methods("GET")
	root.HandleFunc("/myprofile/update", r.onlyUsers(r.handleUpdateMyProfile)).Methods("PUT")

	// show other user's profile
	root.HandleFunc("/profile", r.handleViewUserProfile).Methods("GET").Queries("username", "{username:[a-zA-Z0-9]+}")
	// recycle other user's activity
	root.HandleFunc("/profile/recycle", r.handleRecycleUserActivity).Methods("GET").Queries("userid", "{userid:[a-zA-Z0-9]+}").Headers("X-Requested-With", "XMLHttpRequest")

	root.HandleFunc("/login", r.handleLogin).Methods("POST")
	root.HandleFunc("/signin", r.handleSignin).Methods("POST")
	root.HandleFunc("/logout", r.onlyUsers(r.handleLogout)).Methods("GET")

	// handlers for sections
	section := root.PathPrefix("/{section}").Subrouter()

	section.HandleFunc("/", r.handleViewSection).Methods("GET")
	// create a thread
	section.HandleFunc("/new", r.onlyUsers(r.handleNewThread)).Methods("POST")
	// recycle section threads
	section.HandleFunc("/recycle", r.handleRecycleSection).Methods("GET")

	// handlers for threads
	thread := section.PathPrefix("/{thread}").Subrouter()
	thread.HandleFunc("/", r.handleViewThread).Methods("GET")
	// recycle thread comments
	thread.HandleFunc("/recycle", r.handleRecycleComments).Methods("GET")
	// save thread "/{section}/{thread}/save"
	thread.HandleFunc("/save", r.onlyUsers(r.handleSave)).Methods("POST")
	// undo save thread "/{section}/{thread}/undosave"
	thread.HandleFunc("/undosave", r.onlyUsers(r.handleUndoSave)).Methods("POST")
	// delete thread "/{section}/{thread}/delete"
	thread.HandleFunc("/delete", r.onlyUsers(r.handleDeleteThread)).Methods("DELETE")

	// handlers for comments
	comments := thread.PathPrefix("/comment").Subrouter()
	// get 15 subcomments
	comments.HandleFunc("/", r.handleGetSubcomments).Methods("GET").Headers("X-Requested-With", "XMLHttpRequest").Queries("c_id", "{c_id:[a-zA-Z0-9]+}", "offset", "{offset:[0-9]+}")
	// post a comment
	comments.HandleFunc("/", r.onlyUsers(r.handlePostComment)).Methods("POST")
	// delete a comment "/{section}/{thread}/comment/delete?c_id={c_id}"
	comments.HandleFunc("/delete", r.onlyUsers(r.handleDeleteComment)).Methods("DELETE").Queries("c_id", "{c_id:[a-zA-Z0-9]+}")
	// post a subcomment
	comments.HandleFunc("/", r.onlyUsers(r.handlePostSubcomment)).Methods("POST").Queries("c_id", "{c_id:[a-zA-Z0-9]+}")
	// delete a subcomment
	// "/{section}/{thread}/comment/delete?c_id={c_id}&sc_id={sc_id}"
	comments.HandleFunc("/delete", r.onlyUsers(r.handleDeleteSubcomment)).Methods("DELETE").Queries("c_id", "{c_id:[a-zA-Z0-9]+}", "sc_id", 
		"{sc_id:[a-zA-Z0-9]+}")

	// handlers for upvotes
	upvotes := thread.PathPrefix("/upvote").Subrouter()
	// upvote a thread
	upvotes.HandleFunc("/", r.onlyUsers(r.handleUpvoteThread)).Methods("POST")
	// upvote a comment
	upvotes.HandleFunc("/", r.onlyUsers(r.handleUpvoteComment)).Methods("POST").Queries("c_id", "{c_id:[a-zA-Z0-9]+}")
	// upvote a subcomment
	upvotes.HandleFunc("/", r.onlyUsers(r.handleUpvoteSubcomment)).Methods("POST").Queries("c_id", "{c_id:[a-zA-Z0-9]+}", "sc_id", "{sc_id:[a-zA-Z0-9]+}")

	// handlers for upvote undoings
	undoUpvotes := thread.PathPrefix("/undoupvote").Subrouter()
	// undo upvote on a thread
	undoUpvotes.HandleFunc("/", r.onlyUsers(r.handleUndoUpvoteThread)).Methods("POST")
	// undo upvote on a comment
	undoUpvotes.HandleFunc("/", r.onlyUsers(r.handleUndoUpvoteComment)).Methods("POST").Queries("c_id", "{c_id:[a-zA-Z0-9]+}")
	// undo upvote on a subcomment
	undoUpvotes.HandleFunc("/", r.onlyUsers(r.handleUndoUpvoteSubcomment)).Methods("POST").Queries("c_id", "{c_id:[a-zA-Z0-9]+}", "sc_id", "{sc_id:[a-zA-Z0-9]+}")
}
