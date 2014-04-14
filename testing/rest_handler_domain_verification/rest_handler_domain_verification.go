// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"github.com/rafaeljusto/shelter/config"
	"github.com/rafaeljusto/shelter/model"
	"github.com/rafaeljusto/shelter/net/http/rest/context"
	"github.com/rafaeljusto/shelter/net/http/rest/handler"
	"github.com/rafaeljusto/shelter/net/http/rest/protocol"
	"github.com/rafaeljusto/shelter/testing/utils"
	"net/http"
	"strings"
)

var (
	configFilePath string // Path for the config file with the connection information
)

// RESTHandlerDomainTestConfigFile is a structure to store the test configuration file data
type RESTHandlerDomainVerificationTestConfigFile struct {
	config.Config
	DNSServerPort int
}

func init() {
	utils.TestName = "RESTHandlerDomainVerification"
	flag.StringVar(&configFilePath, "config", "", "Configuration file for RESTHandlerDomainVerification test")
}

func main() {
	flag.Parse()

	var restConfig RESTHandlerDomainVerificationTestConfigFile
	err := utils.ReadConfigFile(configFilePath, &restConfig)
	config.ShelterConfig = restConfig.Config

	if err == utils.ErrConfigFileUndefined {
		fmt.Println(err.Error())
		fmt.Println("Usage:")
		flag.PrintDefaults()
		return

	} else if err != nil {
		utils.Fatalln("Error reading configuration file", err)
	}

	utils.StartDNSServer(restConfig.DNSServerPort, restConfig.Scan.UDPMaxSize)

	scanDomain()
	queryDomain()

	utils.Println("SUCCESS!")
}

func scanDomain() {
	dns.HandleFunc("example.com.br.", func(w dns.ResponseWriter, dnsRequestMessage *dns.Msg) {
		defer w.Close()

		dnsResponseMessage := new(dns.Msg)
		defer w.WriteMsg(dnsResponseMessage)

		dnsResponseMessage = &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Authoritative: true,
			},
			Question: dnsRequestMessage.Question,
			Answer: []dns.RR{
				&dns.SOA{
					Hdr: dns.RR_Header{
						Name:   "example.com.br.",
						Rrtype: dns.TypeSOA,
						Class:  dns.ClassINET,
						Ttl:    86400,
					},
					Ns:      "ns1.example.com.br.",
					Mbox:    "rafael.justo.net.br.",
					Serial:  2013112600,
					Refresh: 86400,
					Retry:   86400,
					Expire:  86400,
					Minttl:  900,
				},
			},
		}
		dnsResponseMessage.SetReply(dnsRequestMessage)

		w.WriteMsg(dnsResponseMessage)
	})

	r, err := http.NewRequest("PUT", "/domain/example.com.br./verification",
		strings.NewReader(`{
      "Nameservers": [
        { "Host": "ns1.example.com.br.", "ipv4": "127.0.0.1" },
        { "Host": "ns2.example.com.br.", "ipv4": "127.0.0.1" }
      ]
    }`))
	if err != nil {
		utils.Fatalln("Error creating the HTTP request", err)
	}

	context, err := context.NewContext(r, nil)
	if err != nil {
		utils.Fatalln("Error creating context", err)
	}

	handler.HandleDomainVerification(r, &context)

	if context.ResponseHTTPStatus != http.StatusOK {
		utils.Fatalln("Error scanning domain",
			errors.New(string(context.ResponseContent)))
	}

	var domainResponse protocol.DomainResponse
	if err := json.Unmarshal(context.ResponseContent, &domainResponse); err != nil {
		utils.Fatalln("Error decoding domain response", err)
	}

	if len(domainResponse.Nameservers) != 2 {
		utils.Fatalln("Wrong number of nameservers", nil)
	}

	if domainResponse.Nameservers[0].LastStatus != model.NameserverStatusToString(model.NameserverStatusOK) {
		utils.Fatalln("Scan did not work for ns1", nil)
	}

	if domainResponse.Nameservers[1].LastStatus != model.NameserverStatusToString(model.NameserverStatusOK) {
		utils.Fatalln("Scan did not work for ns2", nil)
	}
}

func queryDomain() {
	dns.HandleFunc("example.com.br.", func(w dns.ResponseWriter, dnsRequestMessage *dns.Msg) {
		defer w.Close()

		dnsResponseMessage := new(dns.Msg)
		defer w.WriteMsg(dnsResponseMessage)

		dnsResponseMessage = &dns.Msg{
			MsgHdr:   dns.MsgHdr{},
			Question: dnsRequestMessage.Question,
			Answer: []dns.RR{
				&dns.NS{
					Hdr: dns.RR_Header{
						Name:   "example.com.br.",
						Rrtype: dns.TypeNS,
						Class:  dns.ClassINET,
						Ttl:    86400,
					},
					Ns: "a.dns.br.",
				},
				&dns.NS{
					Hdr: dns.RR_Header{
						Name:   "example.com.br.",
						Rrtype: dns.TypeNS,
						Class:  dns.ClassINET,
						Ttl:    86400,
					},
					Ns: "b.dns.br.",
				},
			},
		}
		dnsResponseMessage.SetReply(dnsRequestMessage)

		w.WriteMsg(dnsResponseMessage)
	})

	r, err := http.NewRequest("GET", "/domain/example.com.br./verification", nil)
	if err != nil {
		utils.Fatalln("Error creating the HTTP request", err)
	}

	context, err := context.NewContext(r, nil)
	if err != nil {
		utils.Fatalln("Error creating context", err)
	}

	handler.HandleDomainVerification(r, &context)

	if context.ResponseHTTPStatus != http.StatusOK {
		utils.Fatalln("Error scanning domain",
			errors.New(string(context.ResponseContent)))
	}

	var domainResponse protocol.DomainResponse
	if err := json.Unmarshal(context.ResponseContent, &domainResponse); err != nil {
		utils.Fatalln("Error decoding domain response", err)
	}

	if len(domainResponse.Nameservers) != 2 {
		utils.Fatalln("Wrong number of nameservers", nil)
	}
}