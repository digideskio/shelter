// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

// Package handler store the REST handlers of specific URI
package handler

import (
	"github.com/rafaeljusto/shelter/dao"
	"github.com/rafaeljusto/shelter/log"
	"github.com/rafaeljusto/shelter/model"
	"github.com/rafaeljusto/shelter/net/http/rest/interceptor"
	"github.com/rafaeljusto/shelter/net/http/rest/messages"
	"github.com/rafaeljusto/shelter/net/http/rest/protocol"
	"github.com/trajber/handy"
	"labix.org/v2/mgo"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func init() {
	HandleFunc("/domain/{fqdn}", func() handy.Handler {
		return new(DomainHandler)
	})
}

type DomainHandler struct {
	handy.DefaultHandler
	database        *mgo.Database
	databaseSession *mgo.Session
	domain          model.Domain
	language        *messages.LanguagePack
	DomainName      string                    `param:"fqdn"`
	Request         protocol.DomainRequest    `request:"put"`
	Response        *protocol.DomainResponse  `response:"get"`
	Message         *protocol.MessageResponse `error`
}

func (h *DomainHandler) SetDatabaseSession(session *mgo.Session) {
	h.databaseSession = session
}

func (h *DomainHandler) DatabaseSession() *mgo.Session {
	return h.databaseSession
}

func (h *DomainHandler) SetDatabase(database *mgo.Database) {
	h.database = database
}

func (h *DomainHandler) Database() *mgo.Database {
	return h.database
}

func (h *DomainHandler) SetFQDN(fqdn string) {
	h.DomainName = fqdn
}

func (h *DomainHandler) FQDN() string {
	return h.DomainName
}

func (h *DomainHandler) SetDomain(domain model.Domain) {
	h.domain = domain
}

func (h *DomainHandler) LastModified() time.Time {
	return h.domain.LastModifiedAt
}

func (h *DomainHandler) ETag() string {
	return strconv.Itoa(h.domain.Revision)
}

func (h *DomainHandler) SetLanguage(language *messages.LanguagePack) {
	h.language = language
}

func (h *DomainHandler) Language() *messages.LanguagePack {
	return h.language
}

func (h *DomainHandler) MessageResponse(messageId string, roid string) error {
	var err error
	h.Message, err = protocol.NewMessageResponse(messageId, roid, h.language)
	return err
}

func (h *DomainHandler) Get(w http.ResponseWriter, r *http.Request) {
	h.retrieveDomain(w, r)
}

func (h *DomainHandler) Head(w http.ResponseWriter, r *http.Request) {
	h.retrieveDomain(w, r)
}

// The HEAD method is identical to GET except that the server MUST NOT return a message-
// body in the response. But now the responsability for don't adding the body is from the
// mux while writing the response
func (h *DomainHandler) retrieveDomain(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("ETag", h.ETag())
	w.Header().Add("Last-Modified", h.LastModified().Format(time.RFC1123))
	w.WriteHeader(http.StatusOK)

	domainResponse := protocol.ToDomainResponse(h.domain, true)
	h.Response = &domainResponse
}

func (h *DomainHandler) Put(w http.ResponseWriter, r *http.Request) {
	// We need to set the FQDN in the domain request object because it is sent only in the
	// URI and not in the domain request body to avoid information redudancy
	h.Request.FQDN = h.FQDN()

	var err error
	if h.domain, err = protocol.Merge(h.domain, h.Request); err != nil {
		messageId := ""

		switch err {
		case model.ErrInvalidFQDN:
			messageId = "invalid-fqdn"
		case protocol.ErrInvalidDNSKEY:
			messageId = "invalid-dnskey"
		case protocol.ErrInvalidDSAlgorithm:
			messageId = "invalid-ds-algorithm"
		case protocol.ErrInvalidDSDigestType:
			messageId = "invalid-ds-digest-type"
		case protocol.ErrInvalidIP:
			messageId = "invalid-ip"
		case protocol.ErrInvalidLanguage:
			messageId = "invalid-language"
		}

		if len(messageId) == 0 {
			log.Println("Error while merging domain objects for create or "+
				"update operation. Details:", err)
			w.WriteHeader(http.StatusInternalServerError)

		} else {
			if err := h.MessageResponse(messageId, r.URL.RequestURI()); err == nil {
				w.WriteHeader(http.StatusBadRequest)

			} else {
				log.Println("Error while writing response. Details:", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
		return
	}

	domainDAO := dao.DomainDAO{
		Database: h.Database(),
	}

	if err := domainDAO.Save(&h.domain); err != nil {
		if strings.Index(err.Error(), "duplicate key error index") != -1 {
			if err := h.MessageResponse("conflict", r.URL.RequestURI()); err == nil {
				w.WriteHeader(http.StatusConflict)

			} else {
				log.Println("Error while writing response. Details:", err)
				w.WriteHeader(http.StatusInternalServerError)
			}

		} else {
			log.Println("Error while saving domain object for create or "+
				"update operation. Details:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	w.Header().Add("ETag", h.ETag())
	w.Header().Add("Last-Modified", h.LastModified().Format(time.RFC1123))

	if h.domain.Revision == 1 {
		w.Header().Add("Location", "/domain/"+h.domain.FQDN)
		w.WriteHeader(http.StatusCreated)

	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *DomainHandler) Delete(w http.ResponseWriter, r *http.Request) {
	domainDAO := dao.DomainDAO{
		Database: h.Database(),
	}

	if err := domainDAO.Remove(&h.domain); err != nil {
		log.Println("Error while removing domain object. Details:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DomainHandler) Interceptors() handy.InterceptorChain {
	return handy.NewInterceptorChain().
		Chain(new(interceptor.Permission)).
		Chain(interceptor.NewFQDN(h)).
		Chain(interceptor.NewValidator(h)).
		Chain(interceptor.NewDatabase(h)).
		Chain(interceptor.NewDomain(h)).
		Chain(interceptor.NewCache(h)).
		Chain(interceptor.NewJSONCodec(h))
}
