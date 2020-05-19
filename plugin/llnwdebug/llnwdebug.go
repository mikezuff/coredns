package llnwdebug

import (
	"context"
	"fmt"
	"net"
	"sync"

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

func (ld *LLNWDebug) Name() string { return "llnwdebug" }
