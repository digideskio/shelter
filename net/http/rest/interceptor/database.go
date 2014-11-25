// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

// interceptor add steps to the REST request before calling the handler
package interceptor

import (
	"github.com/rafaeljusto/shelter/config"
	"github.com/rafaeljusto/shelter/database/mongodb"
	"github.com/rafaeljusto/shelter/log"
	"gopkg.in/mgo.v2"
	"net/http"
)

type DatabaseHandler interface {
	SetDBSession(*mgo.Session)
	SetDB(*mgo.Database)
	DBSession() *mgo.Session
}

type Database struct {
	databaseHandler DatabaseHandler
}

func NewDatabase(h DatabaseHandler) *Database {
	return &Database{databaseHandler: h}
}

func (i *Database) Before(w http.ResponseWriter, r *http.Request) {
	database, databaseSession, err := mongodb.Open(
		config.ShelterConfig.Database.URIs,
		config.ShelterConfig.Database.Name,
		config.ShelterConfig.Database.Auth.Enabled,
		config.ShelterConfig.Database.Auth.Username,
		config.ShelterConfig.Database.Auth.Password,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error creating database connection. Details:", err)
		return
	}

	i.databaseHandler.SetDBSession(databaseSession)
	i.databaseHandler.SetDB(database)
}

func (i *Database) After(w http.ResponseWriter, r *http.Request) {
	i.databaseHandler.DBSession().Close()
}
