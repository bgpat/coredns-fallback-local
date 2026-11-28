package fallbacklocal

import (
	"context"
	"net"
	"testing"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/miekg/dns"
)

type staticHandler struct {
	rcode int
	rrs   []dns.RR
}

func (h staticHandler) Name() string { return "static" }

func (h staticHandler) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	m := new(dns.Msg)
	m.SetRcode(r, h.rcode)
	m.Answer = h.rrs
	w.WriteMsg(m)
	return h.rcode, nil
}

func TestName(t *testing.T) {
	f := FallbackLocal{}
	if got := f.Name(); got != "fallback_local" {
		t.Fatalf("Name() = %q, want fallback_local", got)
	}
}

func TestServeDNS_passthroughOnSuccess(t *testing.T) {
	want := &dns.A{
		Hdr: dns.RR_Header{
			Name:   "example.com.",
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		A: net.IPv4(93, 184, 216, 34),
	}
	f := FallbackLocal{Next: staticHandler{rcode: dns.RcodeSuccess, rrs: []dns.RR{want}}}

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)
	rec := dnstest.NewRecorder(&testResponseWriter{})

	rcode, err := f.ServeDNS(context.Background(), rec, req)
	if err != nil {
		t.Fatalf("ServeDNS: %v", err)
	}
	if rcode != dns.RcodeSuccess {
		t.Fatalf("rcode = %d, want success", rcode)
	}
	if len(rec.Msg.Answer) != 1 {
		t.Fatalf("Answer len = %d, want 1", len(rec.Msg.Answer))
	}
	if a, ok := rec.Msg.Answer[0].(*dns.A); !ok || !a.A.Equal(want.A) {
		t.Fatalf("unexpected answer: %#v", rec.Msg.Answer[0])
	}
}

func TestServeDNS_fallbackOnNXDOMAIN(t *testing.T) {
	f := FallbackLocal{Next: staticHandler{rcode: dns.RcodeNameError}}

	req := new(dns.Msg)
	req.SetQuestion("localhost.", dns.TypeA)
	rec := dnstest.NewRecorder(&testResponseWriter{})

	rcode, err := f.ServeDNS(context.Background(), rec, req)
	if err != nil {
		t.Fatalf("ServeDNS: %v", err)
	}
	if rcode != dns.RcodeNameError {
		t.Fatalf("rcode = %d, want NXDOMAIN passthrough", rcode)
	}
	if rec.Msg == nil || rec.Msg.Rcode != dns.RcodeSuccess {
		t.Fatalf("expected successful fallback reply, got %v", rec.Msg)
	}
	if len(rec.Msg.Answer) == 0 {
		t.Fatal("expected A records from getaddrinfo fallback")
	}
}

func TestServeDNS_ignoresNonAQueries(t *testing.T) {
	f := FallbackLocal{Next: staticHandler{rcode: dns.RcodeNameError}}

	req := new(dns.Msg)
	req.SetQuestion("localhost.", dns.TypeAAAA)
	rec := dnstest.NewRecorder(&testResponseWriter{})

	_, err := f.ServeDNS(context.Background(), rec, req)
	if err != nil {
		t.Fatalf("ServeDNS: %v", err)
	}
	if rec.Msg == nil || rec.Msg.Rcode == dns.RcodeSuccess {
		t.Fatal("expected no fallback for AAAA when upstream failed")
	}
}

// testResponseWriter implements dns.ResponseWriter for dnstest.
type testResponseWriter struct{}

func (testResponseWriter) LocalAddr() net.Addr  { return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 53} }
func (testResponseWriter) RemoteAddr() net.Addr { return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345} }
func (testResponseWriter) WriteMsg(*dns.Msg) error { return nil }
func (testResponseWriter) Write([]byte) (int, error) { return len([]byte("ok")), nil }
func (testResponseWriter) Close() error           { return nil }
func (testResponseWriter) TsigStatus() error      { return nil }
func (testResponseWriter) TsigTimersOnly(bool)    {}
func (testResponseWriter) Hijack()                {}

var _ dns.ResponseWriter = testResponseWriter{}
var _ plugin.Handler = staticHandler{}