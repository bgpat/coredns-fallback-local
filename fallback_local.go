package fallbacklocal

import (
	"context"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/nonwriter"
	"github.com/miekg/dns"
)

type FallbackLocal struct {
	Next plugin.Handler
}

func (f FallbackLocal) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	nw := nonwriter.New(w)
	nrcode, err := plugin.NextOrFailure(f.Name(), f.Next, ctx, nw, r)
	msg := nw.Msg
	rcode := nrcode
	if nw.Msg == nil || nw.Msg.Rcode != dns.RcodeSuccess {
		answers := make([]dns.RR, 0)
		for _, q := range r.Question {
			if q.Qclass != dns.ClassINET || q.Qtype != dns.TypeA {
				continue
			}
			host := strings.TrimRight(q.Name, ".")
			addrs, err := net.LookupHost(host)
			if err != nil {
				log.Info(err)
				continue
			}
			for _, addr := range addrs {
				ip := net.ParseIP(addr)
				if ip == nil || ip.IsUnspecified() {
					log.Warningf("failed to parse IP address: %v", addr)
					continue
				}
				answers = append(answers, &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    0,
					},
					A: ip,
				})
			}
		}
		if len(answers) > 0 {
			msg = new(dns.Msg)
			msg.SetReply(r)
			msg.Answer = answers
		}
	}
	w.WriteMsg(msg)
	return rcode, err
}

func (f FallbackLocal) Name() string {
	return "fallback_local"
}
