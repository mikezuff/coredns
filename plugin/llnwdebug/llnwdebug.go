package llnwdebug

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// TODO: get answers from DNS using FQDN (/bin/hostname -f)
var answer4 = net.ParseIP("34.105.111.80").To4()

type LLNWDebug struct {
	lock        sync.Mutex
	dnsRequests map[string]log
}

type log struct {
	lastUpdate time.Time
	log        []RequestInfo
}

type RequestInfo struct {
	Resolver    string `json:"Resolver"`
	EDNS0Subnet string `json:"EDNS0Subnet,omitempty"`
	QType       string `json:"-"`
}

func (ld *LLNWDebug) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	qname := state.QName()
	qtype := state.QType()
	resolver, _, err := net.SplitHostPort(state.RemoteAddr())
	if err != nil {
		fmt.Printf("Error splitting resolver %s\n", state.RemoteAddr())
	}

	// TODO: log for real?
	fmt.Printf("DNS request %s from %s: %s\n", dns.TypeToString[qtype], resolver, qname)
	var edns0Subnet string
	if opt := r.IsEdns0(); opt != nil {
		for _, s := range opt.Option {
			switch e := s.(type) {
			case *dns.EDNS0_SUBNET:
				edns0Subnet = fmt.Sprintf("%s/%d", e.Address.String(), e.SourceNetmask)
				fmt.Printf("\tEDNS0 SUBNET %s\n", edns0Subnet)
			}
		}
	}

	switch qtype {
	case dns.TypeA:
		a := new(dns.Msg)
		a.SetReply(r)
		a.Authoritative = true

		var rr dns.RR
		rr = new(dns.A)
		rr.(*dns.A).Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: state.QClass()}
		rr.(*dns.A).A = answer4

		a.Answer = []dns.RR{rr}

		w.WriteMsg(a)
		ld.recordResolver(qname, resolver, edns0Subnet, dns.TypeToString[qtype])
	}

	return dns.RcodeSuccess, nil
}

func (ld *LLNWDebug) recordResolver(qname, resolver, EDNS0Subnet, qtype string) {
	now := time.Now()

	ld.lock.Lock()
	defer ld.lock.Unlock()

	l := ld.dnsRequests[qname]
	if len(l.log) > 10 {
		return // we're being abused
	}
	l.lastUpdate = now
	l.log = append(l.log, RequestInfo{Resolver: resolver, EDNS0Subnet: EDNS0Subnet, QType: qtype})
	ld.dnsRequests[qname] = l
}

// ServeHTTP responds with information about the DNS resolver.
func (ld *LLNWDebug) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type Response struct {
		Resolvers []RequestInfo `json:"Resolvers,omitempty"`
	}

	ld.lock.Lock()
	defer ld.lock.Unlock()

	var resp Response
	clientAddr, _, _ := net.SplitHostPort(r.RemoteAddr)
	//resp.Client.Addr = clientAddr

	if ri, ok := ld.dnsRequests[r.Host+"."]; ok {
		resp.Resolvers = ri.log
		fmt.Printf("GET from %s %s ResolverInfo: %#v\n", clientAddr, r.Host, ri)
	} else {
		fmt.Printf("GET from %s %s ResolverInfo: unknown\n", clientAddr, r.Host)
	}

	w.Header().Add("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(&resp)
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 4)
	rand.Read(b)
	http.Redirect(w, r, fmt.Sprintf("http://ri-%d-%s.%s/resolverinfo",
		time.Now().Unix(), hex.EncodeToString(b), r.Host), http.StatusFound)
}

func (ld *LLNWDebug) Name() string { return "llnwdebug" }
