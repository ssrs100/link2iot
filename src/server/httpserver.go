package server

import (
	"auth"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
)

type User struct {
	Name      string `json:"name"`
	Password  string `json:"password"`
	ProjectId string `json:"project_id"`
}

func AddUser(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	log.Info("User add start.")
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("Receive body failed: %v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()
	user := &User{}
	err = json.Unmarshal(body, user)
	if err != nil {
		log.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if a := auth.GetAuth(); a != nil {
		userMap := make(map[string]string)
		userMap["name"] = user.Name
		userMap["password"] = user.Password
		userMap["project_id"] = user.ProjectId
		a.AddUser(userMap)
	} else {
		log.Error("auth is nil , add user fail.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	log.Info("User add success.")
}

func DelUser(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	name := ps.ByName("username")
	log.Info("User del start, name:%s.", name)

	if a := auth.GetAuth(); a != nil {
		a.DelUser(name)
	} else {
		log.Error("auth is nil , del user fail.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	log.Info("User del success.")
}

func Health(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

func StartHTTPServer(host, port string) {
	log.Info("Start HTTP Server.")
	router := httprouter.New()

	// Set router options.
	router.HandleMethodNotAllowed = true
	router.HandleOPTIONS = true
	router.RedirectTrailingSlash = true

	// Set the routes for the application.

	// Route for http
	router.GET("/v1/heart", Health)

	router.POST("/v1/users", AddUser)
	router.DELETE("/v1/users/:username", DelUser)

	server := &http.Server{Addr: host + ":" + port, Handler: router}

	log.Info("Starting http server on port %v", port)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	log.Info("Stop http server on port %v", port)
}
