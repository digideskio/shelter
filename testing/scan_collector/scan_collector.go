// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"github.com/rafaeljusto/shelter/Godeps/_workspace/src/gopkg.in/mgo.v2"
	"github.com/rafaeljusto/shelter/dao"
	"github.com/rafaeljusto/shelter/database/mongodb"
	"github.com/rafaeljusto/shelter/model"
	"github.com/rafaeljusto/shelter/net/scan"
	"github.com/rafaeljusto/shelter/testing/utils"
	"net"
	"net/mail"
	"sync"
)

var (
	configFilePath string // Path for the config file with the connection information
)

// ScanCollectorTestConfigFile is a structure to store the test configuration file data
type ScanCollectorTestConfigFile struct {
	Database struct {
		URI  string
		Name string
	}

	Scan struct {
		DomainsBufferSize int // Size of the channels between the querier and the collector
		SaveAtOnce        int // Number of domains to acumulate before saving all togheter
	}
}

func init() {
	utils.TestName = "ScanCollector"
	flag.StringVar(&configFilePath, "config", "", "Configuration file for ScanInjector test")
}

func main() {
	flag.Parse()

	var config ScanCollectorTestConfigFile
	err := utils.ReadConfigFile(configFilePath, &config)

	if err == utils.ErrConfigFileUndefined {
		fmt.Println(err.Error())
		fmt.Println("Usage:")
		flag.PrintDefaults()
		return

	} else if err != nil {
		utils.Fatalln("Error reading configuration file", err)
	}

	database, databaseSession, err := mongodb.Open(
		[]string{config.Database.URI},
		config.Database.Name,
		false, "", "",
	)
	if err != nil {
		utils.Fatalln("Error connecting the database", err)
	}
	defer databaseSession.Close()

	// Remove all data before starting the test. This is necessary because maybe in the last
	// test there was an error and the data wasn't removed from the database
	utils.ClearDatabase(database)

	domainWithErrors(config, database)
	domainWithNoErrors(config, database)

	utils.Println("SUCCESS!")
}

func domainWithErrors(config ScanCollectorTestConfigFile, database *mgo.Database) {
	domainsToSave := make(chan *model.Domain, config.Scan.DomainsBufferSize)
	domainsToSave <- &model.Domain{
		FQDN: "br.",
		Nameservers: []model.Nameserver{
			{
				Host:       "ns1.br",
				IPv4:       net.ParseIP("127.0.0.1"),
				LastStatus: model.NameserverStatusTimeout,
			},
		},
		DSSet: []model.DS{
			{
				Keytag:     1234,
				Algorithm:  model.DSAlgorithmRSASHA1NSEC3,
				DigestType: model.DSDigestTypeSHA1,
				Digest:     "EAA0978F38879DB70A53F9FF1ACF21D046A98B5C",
				LastStatus: model.DSStatusExpiredSignature,
			},
		},
	}
	domainsToSave <- nil

	model.StartNewScan()
	runScan(config, database, domainsToSave)

	domainDAO := dao.DomainDAO{
		Database: database,
	}

	domain, err := domainDAO.FindByFQDN("br.")
	if err != nil {
		utils.Fatalln("Error loading domain with problems", err)
	}

	if len(domain.Nameservers) == 0 {
		utils.Fatalln("Error saving nameservers", nil)
	}

	if domain.Nameservers[0].LastStatus != model.NameserverStatusTimeout {
		utils.Fatalln("Error setting status in the nameserver", nil)
	}

	if len(domain.DSSet) == 0 {
		utils.Fatalln("Error saving the DS set", nil)
	}

	if domain.DSSet[0].LastStatus != model.DSStatusExpiredSignature {
		utils.Fatalln("Error setting status in the DS", nil)
	}

	if err := domainDAO.RemoveByFQDN("br."); err != nil {
		utils.Fatalln("Error removing test domain", err)
	}

	currentScan := model.GetCurrentScan()
	if currentScan.DomainsScanned != 1 || currentScan.DomainsWithDNSSECScanned != 1 {
		utils.Fatalln("Not counting domains for scan progress when there're errors", nil)
	}

	if currentScan.NameserverStatistics[model.NameserverStatusToString(model.NameserverStatusTimeout)] != 1 ||
		currentScan.DSStatistics[model.DSStatusToString(model.DSStatusExpiredSignature)] != 1 {
		utils.Fatalln("Not counting statistics properly when there're errors", nil)
	}
}

func domainWithNoErrors(config ScanCollectorTestConfigFile, database *mgo.Database) {
	domainsToSave := make(chan *model.Domain, config.Scan.DomainsBufferSize)
	domainsToSave <- &model.Domain{
		FQDN: "br.",
		Nameservers: []model.Nameserver{
			{
				Host:       "ns1.br",
				IPv4:       net.ParseIP("127.0.0.1"),
				LastStatus: model.NameserverStatusOK,
			},
		},
		DSSet: []model.DS{
			{
				Keytag:     1234,
				Algorithm:  model.DSAlgorithmRSASHA1NSEC3,
				DigestType: model.DSDigestTypeSHA1,
				Digest:     "EAA0978F38879DB70A53F9FF1ACF21D046A98B5C",
				LastStatus: model.DSStatusOK,
			},
		},
	}
	domainsToSave <- nil

	model.StartNewScan()
	runScan(config, database, domainsToSave)

	domainDAO := dao.DomainDAO{
		Database: database,
	}

	domain, err := domainDAO.FindByFQDN("br.")
	if err != nil {
		utils.Fatalln("Error loading domain with problems", err)
	}

	if len(domain.Nameservers) == 0 {
		utils.Fatalln("Error saving nameservers", nil)
	}

	if domain.Nameservers[0].LastStatus != model.NameserverStatusOK {
		utils.Fatalln("Error setting status in the nameserver", nil)
	}

	if len(domain.DSSet) == 0 {
		utils.Fatalln("Error saving the DS set", nil)
	}

	if domain.DSSet[0].LastStatus != model.DSStatusOK {
		utils.Fatalln("Error setting status in the DS", nil)
	}

	if err := domainDAO.RemoveByFQDN("br."); err != nil {
		utils.Fatalln("Error removing test domain", err)
	}

	currentScan := model.GetCurrentScan()
	if currentScan.DomainsScanned != 1 || currentScan.DomainsWithDNSSECScanned != 1 {
		utils.Fatalln("Not counting domains for scan progress when there're no errors", nil)
	}

	if currentScan.NameserverStatistics[model.NameserverStatusToString(model.NameserverStatusOK)] != 1 ||
		currentScan.DSStatistics[model.DSStatusToString(model.DSStatusOK)] != 1 {
		utils.Fatalln("Not counting statistics properly when there're no errors", nil)
	}
}

// Method responsable to configure and start scan injector for tests
func runScan(config ScanCollectorTestConfigFile,
	database *mgo.Database,
	domainsToSave chan *model.Domain) {

	scanCollector := scan.NewCollector(database, config.Scan.SaveAtOnce)

	var scanGroup sync.WaitGroup
	errorsChannel := make(chan error)
	scanCollector.Start(&scanGroup, domainsToSave, errorsChannel)

	go func() {
		select {
		case err := <-errorsChannel:
			utils.Fatalln("Error saving domain", err)
		}
	}()

	scanGroup.Wait()
}

// Function to mock a domain object
func newDomain() model.Domain {
	var domain model.Domain
	domain.FQDN = "rafael.net.br"

	domain.Nameservers = []model.Nameserver{
		{
			Host: "ns1.rafael.net.br",
			IPv4: net.ParseIP("127.0.0.1"),
			IPv6: net.ParseIP("::1"),
		},
		{
			Host: "ns2.rafael.net.br",
			IPv4: net.ParseIP("127.0.0.2"),
		},
	}

	domain.DSSet = []model.DS{
		{
			Keytag:    1234,
			Algorithm: model.DSAlgorithmRSASHA1,
			Digest:    "A790A11EA430A85DA77245F091891F73AA740483",
		},
	}

	owner, _ := mail.ParseAddress("test@rafael.net.br")
	domain.Owners = []model.Owner{
		{
			Email:    owner,
			Language: "pt-BR",
		},
	}

	return domain
}
