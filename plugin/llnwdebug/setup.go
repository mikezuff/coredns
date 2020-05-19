package llnwdebug

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/reuseport"
)

func init() { plugin.Register("llnwdebug", setup) }

/*
TODO:
type shares data between dns and http
request for main page hits resolverinfo endpoint w/ unique dns name
unique dns request comes in, resolver info is saved under key (for a time)
http request comes in, resolver info is retrieved using Request.Host
*/
func setup(c *caddy.Controller) error {
	ld := &LLNWDebug{
		dnsRequests: make(map[string]string),
	}

	c.OnStartup(func() error {
		ln, err := reuseport.Listen("tcp", ":80")
		if err != nil {
			return err
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
			b := make([]byte, 4)
			rand.Read(b)
			http.Redirect(w, r, fmt.Sprintf("http://ri-%d-%s.ri.zuffs.net/resolverinfo",
				time.Now().Unix(), hex.EncodeToString(b)), http.StatusFound)
		})
		mux.HandleFunc("/resolverinfo", func(w http.ResponseWriter, r *http.Request) {
			type RouteInfo struct {
				Addr string `json:"addr"`
				ASN  string `json:"asn"`
				Org  string `json:"org"`
			}
			type Response struct {
				Resolver RouteInfo `json:"resolver,omitempty"`
				Client   RouteInfo `json:"client"`
			}

			ld.lock.Lock()
			defer ld.lock.Unlock()

			var resp Response
			clientAddr, _, _ := net.SplitHostPort(r.RemoteAddr)
			resp.Client.Addr = clientAddr
			resp.Client.ASN = "unknown"
			resp.Client.Org = "unknown"

			if ri, ok := ld.dnsRequests[r.Host+"."]; ok {
				resp.Resolver.Addr = ri
				resp.Resolver.ASN = "unknown"
				resp.Resolver.Org = "unknown"
				fmt.Printf("GET from %s %s ResolverInfo: %#v\n", clientAddr, r.Host, ri)
			} else {
				fmt.Printf("GET from %s %s ResolverInfo: unknown\n", clientAddr, r.Host)
			}

			w.Header().Add("Content-Type", "application/json")
			enc := json.NewEncoder(w)
			enc.Encode(&resp)

		})
		go func() { http.Serve(ln, mux) }()
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return ld
	})
	return nil
}
