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

var answer4 = net.ParseIP("34.105.111.80").To4()

type LLNWDebug struct {
	lock        sync.Mutex
	dnsRequests map[string]string
}

func (ld *LLNWDebug) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	ld.lock.Lock()
	defer ld.lock.Unlock()

	state := request.Request{W: w, Req: r}

	switch state.QType() {
	case dns.TypeA:
		qname := state.QName()
		resolverIP, _, err := net.SplitHostPort(state.RemoteAddr())
		if err == nil {
			ld.dnsRequests[qname] = resolverIP
			fmt.Printf("DNS A from %s: %s\n", resolverIP, qname)
		}

		a := new(dns.Msg)
		a.SetReply(r)
		a.Authoritative = true

		var rr dns.RR
		rr = new(dns.A)
		rr.(*dns.A).Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: state.QClass()}
		rr.(*dns.A).A = answer4

		a.Answer = []dns.RR{rr}

		w.WriteMsg(a)
	}

	return dns.RcodeSuccess, nil
}

// ServeHTTP responds with information about the DNS resolver.
func (ld *LLNWDebug) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type RouteInfo struct {
		Addr string `json:"address"`
		//ASN  string `json:"asn,omitempty"`
		//Org  string `json:"organization,omitempty"`
	}
	type Response struct {
		Resolver RouteInfo `json:"resolver,omitempty"`
		//Client   RouteInfo `json:"client"`
	}

	ld.lock.Lock()
	defer ld.lock.Unlock()

	var resp Response
	clientAddr, _, _ := net.SplitHostPort(r.RemoteAddr)
	//resp.Client.Addr = clientAddr

	if ri, ok := ld.dnsRequests[r.Host+"."]; ok {
		resp.Resolver.Addr = ri
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
