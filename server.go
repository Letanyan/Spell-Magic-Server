package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

func logger(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		println(r.URL.String())
		fn(w, r)
	}
}

func wrappers(fn http.HandlerFunc) http.HandlerFunc {
	if isDebug {
		return logger(fn)
	} else {
		return fn
	}
} 

func serverInit() {
	mux := http.NewServeMux()
	mux.Handle("GET /resources/", http.StripPrefix("/resources", http.FileServer(http.Dir("./resources/"))))

	// UI
	// mux.HandleFunc("GET /api/v1/ui/agent_sign_in_form/{$}", wrappers(componentToRenderer(uiSignInForm())))
	// mux.HandleFunc("GET /api/v1/ui/agent_sign_up_form/{$}", wrappers(componentToRenderer(uiSignUpForm())))
	// mux.HandleFunc("GET /api/v1/ui/agent_booking_form/{$}", wrappers(componentToRenderer(uiAgentBookingForm())))
	

	// User
    mux.HandleFunc("POST /api/v1/user/sign_up/{$}", wrappers(apiUserSignUp))
    mux.HandleFunc("POST /api/v1/user/sign_in/{$}", wrappers(apiUserSignIn))
    mux.HandleFunc("POST /api/v1/user/sign_out/{$}", wrappers(apiUserSignOut))
	// Levels
	mux.HandleFunc("POST /api/v1/level/{$}", wrappers(apiAddLevel))
	mux.HandleFunc("GET /api/v1/level/{$}", wrappers(apiGetLevel))
	mux.HandleFunc("PUT /api/v1/level/{$}", wrappers(apiPutLevel))
	mux.HandleFunc("GET /api/v1/levels/{$}", wrappers(apiGetLevels))
	// LevelUserData
	mux.HandleFunc("POST /api/v1/level/data/{$}", wrappers(apiAddLevelUserData))
	mux.HandleFunc("PUT /api/v1/level/data/{$}", wrappers(apiPutLevelUserData))
	mux.HandleFunc("PUT /api/v1/level/data/start/{$}", wrappers(apiBeginPlayLevel))
	mux.HandleFunc("PUT /api/v1/level/data/save/{$}", wrappers(apiSavePlayLevel))

	mux.HandleFunc("GET /ping/{$}", wrappers(apiPing))


	err := http.ListenAndServe("127.0.0.1:8181", mux)
	if didFail(err) {
		log.Fatal(err)
	}
}

func apiPing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Pong"))
}

// ----------------------------------------------------------------------
// -- User 
// ----------------------------------------------------------------------

func apiUserSignUp(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	password := r.FormValue("password")
	_, session_token, err := dbCreateUser(mainDB, name, password)
	if didFail(err) {
		// TODO: handle error
		return
	}
	w.Header().Add("Set-Cookie", fmt.Sprintf("user_token=%s; SameSite=Strict; Path=/", session_token))
}

func apiUserSignIn(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	password := r.FormValue("password")
	
	_, session_token, err := dbSignInUser(mainDB, name, password)
	if err == nil {
		w.Header().Set("Kind", "SignIn")
		w.Header().Set("Set-Cookie", fmt.Sprintf("user_token=%s; SameSite=Strict; Path=/", session_token))
		w.Write([]byte(session_token))
		return
	}
	_, session_token, err = dbCreateUser(mainDB, name, password)
	if didFail(err) {
		// TODO: handle error
		return
	}
	w.Header().Set("Kind", "SignIn")
	w.Header().Set("Set-Cookie", fmt.Sprintf("user_token=%s; SameSite=Strict; Path=/", session_token))
	w.Write([]byte(session_token))
}

func apiValidSessionToken(r *http.Request) User {
	session := r.CookiesNamed("user_token")
	user := User{ id: -1 }
	if len(session) > 0 {
		user = dbDoesUserHaveASession(mainDB, session[0].Value)
	}
	return user
}

func apiUserSignOut(w http.ResponseWriter, r *http.Request) {
	user := apiValidSessionToken(r)
	if user.id == -1 {
		// TODO: handle error
		return 
	}
	session := r.CookiesNamed("user_token")
	dbSignOutUserSession(mainDB, session[0].Value)
	w.Header().Add("Set-Cookie", "user_token=deleted; expires=Thu, 01 Jan 1970 00:00:00 GMT; SameSite=Strict; Path=/")
}

// ----------------------------------------------------------------------
// -- Levels 
// ----------------------------------------------------------------------

func apiGetLevel(w http.ResponseWriter, r *http.Request) {
	user := apiValidSessionToken(r)
	if user.id == -1 {
		// TODO: handle error
		return 
	}
	levelIdString := r.URL.Query().Get("id")
	levelId, err := strconv.Atoi(levelIdString)
	if err != nil {
		return
	}
	level := dbGetLevel(mainDB, int64(levelId), user.id)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Kind", "GetLevel")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(level)
}

func apiAddLevel(w http.ResponseWriter, r *http.Request) {
	user := apiValidSessionToken(r)
	if user.id == -1 {
		// TODO: handle error
		return 
	} 
	data, _ := io.ReadAll(r.Body)
	name := r.URL.Query().Get("name")
	desc := r.URL.Query().Get("desc")
	_, id, e := dbCreateLevel(mainDB, name, desc, user.id, data)
	if didFail(e, "could not create level") {
		return
	}
	w.Header().Set("Kind", "AddLevel")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(strconv.Itoa(int(id))))
}

func apiPutLevel(w http.ResponseWriter, r *http.Request) {
	user := apiValidSessionToken(r)
	if user.id == -1 {
		// TODO: handle error
		return 
	}
	data, _ := io.ReadAll(r.Body)
	desc := r.URL.Query().Get("desc")
	levelIdString := r.URL.Query().Get("id")
	levelId, err := strconv.Atoi(levelIdString)
	if err != nil {
		return
	}
	e := dbUpdateLevel(mainDB, user, int64(levelId), desc, data)
	if didFail(e, "could not create level") {
		return
	}
	w.Header().Set("Kind", "PutLevel")
	w.WriteHeader(http.StatusCreated)
}

func apiGetLevels(w http.ResponseWriter, r *http.Request) {
	levelLimitString := r.URL.Query().Get("limit")
	levelPageString := r.URL.Query().Get("page")
	levelUserString := r.URL.Query().Get("user")
	levelNameString := r.URL.Query().Get("name")
	levelLimit, err := strconv.Atoi(levelLimitString)
	if err != nil {
		return
	}
	levelPage, err := strconv.Atoi(levelPageString)
	if err != nil {
		return
	}
	var levels []LevelItem
	var new_page int
	if len(levelNameString) > 0 {
		levels, new_page = dbGetLevelsWithName(mainDB, levelNameString, LevelSortRecent, levelPage, levelLimit)
	} else if len(levelUserString) > 0 {
		levels, new_page = dbGetLevelsFromUser(mainDB, levelUserString, LevelSortRecent, levelPage, levelLimit)
	} else {
		levels, new_page = dbGetLevels(mainDB, LevelSortRecent, levelPage, levelLimit)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Kind", "GetLevels")
	w.Header().Set("Page", strconv.Itoa(new_page))
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(levels)
}

func apiAddLevelUserData(w http.ResponseWriter, r *http.Request) {
	user := apiValidSessionToken(r)
	if user.id == -1 {
		// TODO: handle error
		return 
	}
	levelIdString := r.URL.Query().Get("levelId")
	levelId, err := strconv.Atoi(levelIdString)
	if err != nil {
		return
	}
	e := dbCreateLevelUserData(mainDB, int64(levelId), user.id)
	if didFail(e, "could not create level user data") {
		return
	}
	w.Header().Set("Kind", "AddLevelUserData")
	w.WriteHeader(http.StatusCreated)
}

func apiPutLevelUserData(w http.ResponseWriter, r *http.Request) {
	user := apiValidSessionToken(r)
	if user.id == -1 {
		// TODO: handle error
		return 
	}
	levelIdString := r.URL.Query().Get("levelId")
	levelId, err := strconv.Atoi(levelIdString)
	if err != nil {
		return
	}
	voteString := r.URL.Query().Get("vote")
	voteAmount := 0
	if voteString == "upvote" {
		voteAmount = 1
	} else if voteString == "downvote" {
		voteAmount = -1
	}

	e := dbUpdateLevelUserData(mainDB, int64(levelId), user.id, int64(voteAmount))
	if didFail(e, "could not update level user data") {
		return
	}
	w.Header().Set("Kind", "PutLevelUserData")
	w.WriteHeader(http.StatusCreated)
}

func apiBeginPlayLevel(w http.ResponseWriter, r *http.Request) {
	user := apiValidSessionToken(r)
	if user.id == -1 {
		// TODO: handle error
		return 
	}
	levelIdString := r.URL.Query().Get("levelId")
	levelId, err := strconv.Atoi(levelIdString)
	if err != nil {
		return
	}

	e := dbBeginLevelUserDataPlaytime(mainDB, int64(levelId), user.id)
	if didFail(e, "could not begin level user data start play") {
		return
	}
	w.Header().Set("Kind", "PutLevelUserData")
	w.WriteHeader(http.StatusCreated)
}

func apiSavePlayLevel(w http.ResponseWriter, r *http.Request) {
	user := apiValidSessionToken(r)
	if user.id == -1 {
		// TODO: handle error
		return 
	}
	levelIdString := r.URL.Query().Get("levelId")
	levelId, err := strconv.Atoi(levelIdString)
	if err != nil {
		return
	}

	e := dbSaveLevelUserDataPlaytime(mainDB, int64(levelId), user.id)
	if didFail(e, "could not begin level user data start play") {
		return
	}
	w.Header().Set("Kind", "PutLevelUserData")
	w.WriteHeader(http.StatusCreated)
}