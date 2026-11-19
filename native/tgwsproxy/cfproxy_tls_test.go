package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestWebSocketTLSNameDoesNotChangeHTTPHost(t *testing.T) {
	for _, tc := range []struct{ host, sni string }{
		{"kws2.pool.example", "pool.example"},
		{"kws4.pool.example", "pool.example"},
		{"kws2.web.telegram.org", ""},
		{"worker.example", ""},
	} {
		t.Run(tc.host, func(t *testing.T) {
			got := make(chan [3]string, 1)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got <- [3]string{r.TLS.ServerName, r.Host, r.URL.Path}
				w.Header().Set("Connection", "Upgrade")
				w.Header().Set("Upgrade", "websocket")
				w.WriteHeader(http.StatusSwitchingProtocols)
			}))
			defer server.Close()
			ws, err := wsConnectWithCurvesContextDial(context.Background(), "target", tc.host, tc.sni, "/apiws", 2, classicalTLSCurves(),
				func(ctx context.Context, network, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
				})
			if err != nil {
				t.Fatal(err)
			}
			defer ws.Close()
			wantSNI := tc.sni
			if wantSNI == "" {
				wantSNI = tc.host
			}
			if actual := <-got; actual != [3]string{wantSNI, tc.host, "/apiws"} {
				t.Fatalf("SNI/Host/path=%v", actual)
			}
		})
	}
}

func TestCFProxyTLSClientHelloUsesClassicalGroups(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	got := make(chan []tls.CurveID, 1)
	serverTLS := tls.Server(server, &tls.Config{GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
		got <- append([]tls.CurveID(nil), hello.SupportedCurves...)
		return nil, errors.New("captured ClientHello")
	}})
	go serverTLS.Handshake()
	config := webSocketTLSConfig("example.test", classicalTLSCurves())
	_ = tls.Client(client, config).Handshake()
	curves := <-got
	if !reflect.DeepEqual(curves, []tls.CurveID{tls.X25519, tls.CurveP256}) {
		t.Fatalf("CF ClientHello groups=%v", curves)
	}
	if webSocketTLSConfig("example.test", nil).CurvePreferences != nil {
		t.Fatal("other WebSocket routes must retain Go's default groups")
	}
}

func TestCFProxyTLSFallbackIsBoundedPerDomain(t *testing.T) {
	primaryErr := &wsStageError{Stage: "tls_handshake", Err: errors.New("handshake timeout")}
	var calls [][]tls.CurveID
	d := compatibleWebSocketTLSDialer{dial: func(_, _, _ string, _ float64, curves []tls.CurveID) (*RawWebSocket, error) {
		calls = append(calls, curves)
		if curves == nil {
			return nil, &wsStageError{Stage: "tcp_dial", Err: errors.New("offline")}
		}
		return nil, primaryErr
	}}
	for _, ip := range []string{"first", "second", "third"} {
		if _, err := d.connect(ip, "example.test", "/apiws", 1); !errors.Is(err, primaryErr) {
			t.Fatalf("lost original failure: %v", err)
		}
	}
	if len(calls) != 4 || calls[0] == nil || calls[1] != nil || calls[2] == nil || calls[3] == nil {
		t.Fatalf("unexpected TLS profile sequence: %v", calls)
	}
}

func TestCFProxyTLSFallbackSuccess(t *testing.T) {
	socket := &RawWebSocket{}
	count := 0
	d := compatibleWebSocketTLSDialer{dial: func(_, _, _ string, _ float64, curves []tls.CurveID) (*RawWebSocket, error) {
		count++
		if curves != nil {
			return nil, &wsStageError{Stage: "tls_handshake", Err: errors.New("handshake failed")}
		}
		return socket, nil
	}}
	got, err := d.connect("target", "example.test", "/apiws", 1)
	if err != nil || got != socket || count != 2 {
		t.Fatalf("socket=%v err=%v calls=%d", got, err, count)
	}
}

func TestCFProxyTLSDoesNotRetryNonTLSFailures(t *testing.T) {
	for _, failure := range []error{
		&wsStageError{Stage: "tcp_dial", Err: errors.New("offline")},
		&wsStageError{Stage: "ws_upgrade_read", Err: errors.New("timeout")},
		&wsStageError{Stage: "tls_handshake", Err: context.Canceled},
		&WsHandshakeError{StatusCode: 503},
	} {
		calls := 0
		d := compatibleWebSocketTLSDialer{dial: func(_, _, _ string, _ float64, _ []tls.CurveID) (*RawWebSocket, error) {
			calls++
			return nil, failure
		}}
		_, err := d.connect("target", "example.test", "/apiws", 1)
		if !errors.Is(err, failure) || calls != 1 {
			t.Fatalf("failure=%v err=%v calls=%d", failure, err, calls)
		}
	}
}

func TestCFProxyTLSDoesNotStartFallbackAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	d := compatibleWebSocketTLSDialer{ctx: ctx, dial: func(_, _, _ string, _ float64, _ []tls.CurveID) (*RawWebSocket, error) {
		calls++
		cancel()
		return nil, &wsStageError{Stage: "tls_handshake", Err: errors.New("handshake timeout")}
	}}
	_, err := d.connect("target", "example.test", "/apiws", 1)
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestPlainCFWebSocketUsesPort80AndKeepsUpgradeHeaders(t *testing.T) {
	got := make(chan [3]string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- [3]string{r.Host, r.URL.Path, r.Header.Get("Sec-WebSocket-Protocol")}
		w.Header().Set("Upgrade", "websocket")
		w.Header().Set("Connection", "Upgrade")
		w.WriteHeader(http.StatusSwitchingProtocols)
	}))
	defer server.Close()
	var dialAddress string
	ws, err := wsConnectWithTLSConfigContextDial(context.Background(), "target", "kws2.pool.example", "/apiws", 2, nil,
		func(ctx context.Context, network, address string) (net.Conn, error) {
			dialAddress = address
			return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
		})
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if dialAddress != "target:80" {
		t.Fatalf("dialed %s, want port 80", dialAddress)
	}
	if actual := <-got; actual != [3]string{"kws2.pool.example", "/apiws", "binary"} {
		t.Fatalf("upgrade=%v", actual)
	}
}

func TestCFNoTLSRequiresExplicitRuntimeSetting(t *testing.T) {
	for _, tc := range []struct {
		config string
		plain  bool
	}{
		{"", false}, {"@cfproxy_no_tls=0", false}, {"@cfproxy_no_tls=1", true},
	} {
		_, settings, err := parseRuntimeConfig(tc.config)
		if err != nil || settings.CF.NoTLS != tc.plain {
			t.Fatalf("config=%q plain=%v err=%v", tc.config, settings.CF.NoTLS, err)
		}
	}
}
