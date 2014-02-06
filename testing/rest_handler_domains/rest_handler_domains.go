package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/rafaeljusto/shelter/database/mongodb"
	"github.com/rafaeljusto/shelter/net/http/rest/context"
	"github.com/rafaeljusto/shelter/net/http/rest/handler"
	"github.com/rafaeljusto/shelter/net/http/rest/protocol"
	"github.com/rafaeljusto/shelter/testing/utils"
	"labix.org/v2/mgo"
	"net/http"
	"strings"
)

var (
	configFilePath string // Path for the config file with the connection information
)

// RESTHandlerDomainsTestConfigFile is a structure to store the test configuration file data
type RESTHandlerDomainsTestConfigFile struct {
	Database struct {
		URI  string
		Name string
	}
}

func init() {
	utils.TestName = "RESTHandlerDomains"
	flag.StringVar(&configFilePath, "config", "", "Configuration file for RESTHandlerDomains test")
}

func main() {
	flag.Parse()

	var config RESTHandlerDomainsTestConfigFile
	err := utils.ReadConfigFile(configFilePath, &config)

	if err == utils.ErrConfigFileUndefined {
		fmt.Println(err.Error())
		fmt.Println("Usage:")
		flag.PrintDefaults()
		return

	} else if err != nil {
		utils.Fatalln("Error reading configuration file", err)
	}

	database, databaseSession, err := mongodb.Open(config.Database.URI, config.Database.Name)
	if err != nil {
		utils.Fatalln("Error connecting the database", err)
	}
	defer databaseSession.Close()

	// If there was some problem in the last test, there could be some data in the
	// database, so let's clear it to don't affect this test. We avoid checking the error,
	// because if the collection does not exist yet, it will be created in the first
	// insert
	utils.ClearDatabase(database)

	createDomains(database)
	retrieveDomains(database)
	retrieveDomainsMetadata(database)
	deleteDomains(database)

	utils.Println("SUCCESS!")
}

func createDomains(database *mgo.Database) {
	for i := 0; i < 100; i++ {
		r, err := http.NewRequest("PUT", fmt.Sprintf("/domain/example%d.com.br.", i),
			strings.NewReader(`{
      "Nameservers": [
        { "Host": "ns1.example.com.br." },
        { "Host": "ns2.example.com.br." }
      ],
      "Owners": [
        "admin@example.com.br."
      ]
    }`))
		if err != nil {
			utils.Fatalln("Error creting the HTTP request", err)
		}

		context, err := context.NewContext(r, database)
		if err != nil {
			utils.Fatalln("Error creating context", err)
		}

		handler.HandleDomain(r, &context)

		if context.ResponseHTTPStatus != http.StatusCreated {
			utils.Fatalln("Error creating domain",
				errors.New(string(context.ResponseContent)))
		}
	}
}

func retrieveDomains(database *mgo.Database) {
	data := []struct {
		URI                string
		ExpectedHTTPStatus int
		ContentCheck       func([]byte)
	}{
		{
			URI:                "/domains/?orderby=xxx:desc&pagesize=10&page=1",
			ExpectedHTTPStatus: http.StatusBadRequest,
		},
		{
			URI:                "/domains/?orderby=fqdn:xxx&pagesize=10&page=1",
			ExpectedHTTPStatus: http.StatusBadRequest,
		},
		{
			URI:                "/domains/?orderby=fqdn:desc&pagesize=xxx&page=1",
			ExpectedHTTPStatus: http.StatusBadRequest,
		},
		{
			URI:                "/domains/?orderby=fqdn:desc&pagesize=10&page=xxx",
			ExpectedHTTPStatus: http.StatusBadRequest,
		},
		{
			URI:                "/domains/?orderby=fqdn:desc&pagesize=10&page=1",
			ExpectedHTTPStatus: http.StatusOK,
			ContentCheck: func(content []byte) {
				var domainsResponse protocol.DomainsResponse
				json.Unmarshal(content, &domainsResponse)

				if len(domainsResponse.Domains) != 10 {
					utils.Fatalln("Error retrieving the wrong number of domains", nil)
				}
			},
		},
	}

	for _, item := range data {
		r, err := http.NewRequest("GET", item.URI, nil)
		if err != nil {
			utils.Fatalln("Error creting the HTTP request", err)
		}

		context, err := context.NewContext(r, database)
		if err != nil {
			utils.Fatalln("Error creating context", err)
		}

		handler.HandleDomains(r, &context)

		if context.ResponseHTTPStatus != item.ExpectedHTTPStatus {
			utils.Fatalln(fmt.Sprintf("Error when requesting domains using the URI [%s]. "+
				"Expected HTTP status code %d but got %d", item.URI,
				item.ExpectedHTTPStatus, context.ResponseHTTPStatus), nil)
		}

		if item.ContentCheck != nil {
			item.ContentCheck(context.ResponseContent)
		}
	}
}

func retrieveDomainsMetadata(database *mgo.Database) {
	r, err := http.NewRequest("GET", "/domains/?orderby=fqdn:desc&pagesize=10&page=1", nil)
	if err != nil {
		utils.Fatalln("Error creting the HTTP request", err)
	}

	context1, err := context.NewContext(r, database)
	if err != nil {
		utils.Fatalln("Error creating context", err)
	}

	handler.HandleDomains(r, &context1)

	if context1.ResponseHTTPStatus != http.StatusOK {
		utils.Fatalln("Error retrieving domains",
			errors.New(string(context1.ResponseContent)))
	}

	r, err = http.NewRequest("HEAD", "/domains/?orderby=fqdn:desc&pagesize=10&page=1", nil)
	if err != nil {
		utils.Fatalln("Error creting the HTTP request", err)
	}

	context2, err := context.NewContext(r, database)
	if err != nil {
		utils.Fatalln("Error creating context", err)
	}

	handler.HandleDomains(r, &context2)

	if context2.ResponseHTTPStatus != http.StatusOK {
		utils.Fatalln("Error retrieving domains",
			errors.New(string(context2.ResponseContent)))
	}

	if string(context1.ResponseContent) != string(context2.ResponseContent) {
		utils.Fatalln("At this point the GET and HEAD method should "+
			"return the same body content", nil)
	}
}

func deleteDomains(database *mgo.Database) {
	for i := 0; i < 100; i++ {
		r, err := http.NewRequest("DELETE", fmt.Sprintf("/domain/example%d.com.br.", i), nil)
		if err != nil {
			utils.Fatalln("Error creting the HTTP request", err)
		}

		context, err := context.NewContext(r, database)
		if err != nil {
			utils.Fatalln("Error creating context", err)
		}

		handler.HandleDomain(r, &context)

		if context.ResponseHTTPStatus != http.StatusNoContent {
			utils.Fatalln("Error removing domain",
				errors.New(string(context.ResponseContent)))
		}
	}
}
