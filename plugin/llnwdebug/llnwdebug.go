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

	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

type LLNWDebug struct {
	lock        sync.Mutex
	dnsRequests map[string]log
	answers4    []net.IP
	answers6    []net.IP
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

// metadataKeyECS can be used for logging EDNS0 Client Subnet
const metadataKeyECS = "llnwdebug/edns0subnet"

func (ld *LLNWDebug) Metadata(ctx context.Context, state request.Request) context.Context {
	var ecs string

	metadata.SetValueFunc(ctx, metadataKeyECS, func() string {
		if ecs == "" {
			ecs = edns0Subnet(state.Req)
		}
		return ecs
	})
	return ctx
}

func edns0Subnet(r *dns.Msg) string {
	if opt := r.IsEdns0(); opt != nil {
		for _, s := range opt.Option {
			switch e := s.(type) {
			case *dns.EDNS0_SUBNET:
				return fmt.Sprintf("%s/%d", e.Address.String(), e.SourceNetmask)
			}
		}
	}
	return "-"
}

func (ld *LLNWDebug) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	qname := state.QName()
	qtype := state.QType()
	resolver, _, err := net.SplitHostPort(state.RemoteAddr())
	if err != nil {
		fmt.Printf("Error splitting resolver %s\n", state.RemoteAddr())
	}

	// must getECS here, it's unavailable later
	ecs := getECS(ctx)

	switch qtype {
	case dns.TypeA, dns.TypeAAAA:
		a := new(dns.Msg)
		a.SetReply(r)
		a.Authoritative = true

		switch qtype {
		case dns.TypeA:
			for _, a4 := range ld.answers4 {
				var rr dns.RR
				rr = new(dns.A)
				rr.(*dns.A).Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: state.QClass()}
				rr.(*dns.A).A = a4
				a.Answer = append(a.Answer, rr)
			}
		case dns.TypeAAAA:
			for _, a6 := range ld.answers6 {
				var rr dns.RR
				rr = new(dns.AAAA)
				rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeAAAA, Class: state.QClass()}
				rr.(*dns.AAAA).AAAA = a6
				a.Answer = append(a.Answer, rr)
			}
		}

		w.WriteMsg(a)
		if ecs == "-" {
			// for logging, getECS() returns "-" but it should be omitted from the JSON if not present.
			ecs = ""
		}
		ld.recordResolver(qname, resolver, ecs, dns.TypeToString[qtype])
	}

	return dns.RcodeSuccess, nil
}

func getECS(ctx context.Context) string {
	f := metadata.ValueFunc(ctx, metadataKeyECS)
	if f == nil {
		return ""
	}
	return f()
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
		fmt.Printf("[llnwdebug-http] %s %s %v\n", clientAddr, r.Host, ri.log)
	} else {
		fmt.Printf("[llnwdebug-http] %s %s unknown\n", clientAddr, r.Host)
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
