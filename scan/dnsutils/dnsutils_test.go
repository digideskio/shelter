package dnsutils

import (
	"github.com/miekg/dns"
	"testing"
)

func TestFilterRRs(t *testing.T) {
	rrs := []dns.RR{
		&dns.SOA{
			Hdr: dns.RR_Header{
				Rrtype: dns.TypeSOA,
			},
		},
		&dns.DNSKEY{
			Hdr: dns.RR_Header{
				Rrtype: dns.TypeDNSKEY,
			},
		},
		&dns.DNSKEY{
			Hdr: dns.RR_Header{
				Rrtype: dns.TypeDNSKEY,
			},
		},
	}

	filteredRRs := FilterRRs(rrs, dns.TypeDNSKEY)
	for _, rr := range filteredRRs {
		if _, ok := rr.(*dns.DNSKEY); !ok {
			t.Error("Not filtering RRs")
			break
		}
	}
}

func TestFilterFirstRR(t *testing.T) {
	rrs := []dns.RR{
		&dns.SOA{
			Hdr: dns.RR_Header{
				Rrtype: dns.TypeSOA,
			},
		},
		&dns.DNSKEY{
			Hdr: dns.RR_Header{
				Name:   "teste1.com.br.",
				Rrtype: dns.TypeDNSKEY,
			},
		},
		&dns.DNSKEY{
			Hdr: dns.RR_Header{
				Name:   "teste2.com.br.",
				Rrtype: dns.TypeDNSKEY,
			},
		},
	}

	filteredRR := FilterFirstRR(rrs, dns.TypeDNSKEY)
	if filteredRR == nil {
		t.Fatal("Couldn't find a valid RR")
	}

	if dnskey, ok := filteredRR.(*dns.DNSKEY); !ok {
		t.Error("Filtering the wrong resource record")
	} else if dnskey.Header().Name != "teste1.com.br." {
		t.Error("Not returning the first RR of the given type")
	}
}
