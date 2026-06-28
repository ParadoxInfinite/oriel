package server

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHeaderHasToken(t *testing.T) {
	cases := map[string]bool{
		"Upgrade":             true,
		"keep-alive, Upgrade": true,
		"keep-alive,upgrade":  true,
		"keep-alive":          false,
		"upgradex":            false,
	}
	for header, want := range cases {
		if got := headerHasToken(header, "upgrade"); got != want {
			t.Errorf("headerHasToken(%q) = %v, want %v", header, got, want)
		}
	}
}

// maskedFrame builds a single final client→server frame (always masked, per RFC
// 6455). Payloads in this test are short, so only the 7-bit length form is used.
func maskedFrame(opcode byte, payload []byte) []byte {
	mask := []byte{0x12, 0x34, 0x56, 0x78}
	f := []byte{0x80 | opcode, 0x80 | byte(len(payload))}
	f = append(f, mask...)
	for i, c := range payload {
		f = append(f, c^mask[i%4])
	}
	return f
}

// readServerFrame reads one short, unmasked server→client frame.
func readServerFrame(t *testing.T, br *bufio.Reader) (opcode byte, payload []byte) {
	t.Helper()
	var h [2]byte
	if _, err := io.ReadFull(br, h[:]); err != nil {
		t.Fatalf("read frame header: %v", err)
	}
	opcode = h[0] & 0x0f
	n := int(h[1] & 0x7f)
	payload = make([]byte, n)
	if _, err := io.ReadFull(br, payload); err != nil {
		t.Fatalf("read frame payload: %v", err)
	}
	return opcode, payload
}

// TestWebSocketHandshakeAndEcho drives the real handshake + frame codec end to
// end: a raw TCP client performs the upgrade, sends a masked binary frame, and
// the server (which echoes via ReadMessage/WriteBinary) returns it unmasked.
func TestWebSocketHandshakeAndEcho(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := wsUpgrade(w, r)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer ws.Close()
		op, data, err := ws.ReadMessage()
		if err != nil {
			t.Errorf("read: %v", err)
			return
		}
		if op != wsBinary {
			t.Errorf("opcode = %d, want binary", op)
		}
		_ = ws.WriteBinary(data)
	}))
	defer srv.Close()

	conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// RFC 6455 sample key → known accept value.
	const key = "dGhlIHNhbXBsZSBub25jZQ=="
	const wantAccept = "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	req := "GET / HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Version: 13\r\nSec-WebSocket-Key: " + key + "\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	br := bufio.NewReader(conn)
	status, err := br.ReadString('\n')
	if err != nil || !strings.Contains(status, "101") {
		t.Fatalf("handshake status = %q (err %v), want 101", status, err)
	}
	var gotAccept string
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatalf("read headers: %v", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
		if k, v, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(k), "Sec-WebSocket-Accept") {
			gotAccept = strings.TrimSpace(v)
		}
	}
	if gotAccept != wantAccept {
		t.Fatalf("accept = %q, want %q", gotAccept, wantAccept)
	}

	if _, err := conn.Write(maskedFrame(wsBinary, []byte("hello shell"))); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	op, payload := readServerFrame(t, br)
	if op != wsBinary || string(payload) != "hello shell" {
		t.Fatalf("echo = op %d %q, want binary %q", op, payload, "hello shell")
	}
}
