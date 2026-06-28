package server

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// A small, self-contained WebSocket server (RFC 6455) for the one place Oriel
// needs a bidirectional channel: the container shell. The protocol is frozen, so
// this stays put rather than pulling in (and tracking the license/upkeep of) a
// dependency. Scope is deliberately narrow — text/binary data frames plus the
// ping/pong/close control frames — which is all the shell uses.

// RFC 6455 §1.3: the server's accept key is the client key concatenated with this
// fixed GUID, SHA-1'd, then base64-encoded.
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

const (
	wsContinuation = 0x0
	wsText         = 0x1
	wsBinary       = 0x2
	wsClose        = 0x8
	wsPing         = 0x9
	wsPong         = 0xA
)

// wsMaxMessage caps a single inbound message so a client can't make us buffer
// without bound. Shell input (keystrokes, a paste, a resize JSON) is tiny.
const wsMaxMessage = 1 << 20 // 1 MiB

type wsConn struct {
	conn net.Conn
	br   *bufio.Reader
	wmu  sync.Mutex // serialises writes (the output pump and control frames race)
}

// wsUpgrade completes the handshake and hijacks the connection. It returns an
// error (without writing 101) if the request isn't a valid WebSocket upgrade.
func wsUpgrade(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	if !headerHasToken(r.Header.Get("Connection"), "upgrade") || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, fmt.Errorf("not a websocket upgrade")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return nil, fmt.Errorf("unsupported websocket version")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, fmt.Errorf("missing Sec-WebSocket-Key")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("connection does not support hijacking")
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	sum := sha1.Sum([]byte(key + wsGUID))
	accept := base64.StdEncoding.EncodeToString(sum[:])
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := io.WriteString(conn, resp); err != nil {
		conn.Close()
		return nil, err
	}
	return &wsConn{conn: conn, br: rw.Reader}, nil
}

// headerHasToken reports whether a comma-separated header value contains token
// (case-insensitive), e.g. Connection: keep-alive, Upgrade.
func headerHasToken(header, token string) bool {
	for _, part := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}

// ReadMessage returns the next data message (text or binary), transparently
// answering pings and reassembling fragments. It returns io.EOF on a close frame
// or a closed connection.
func (c *wsConn) ReadMessage() (opcode byte, payload []byte, err error) {
	var msg []byte
	var msgOp byte
	for {
		fin, op, data, err := c.readFrame()
		if err != nil {
			return 0, nil, err
		}
		switch op {
		case wsPing:
			if err := c.write(wsPong, data); err != nil {
				return 0, nil, err
			}
			continue
		case wsPong:
			continue
		case wsClose:
			_ = c.write(wsClose, nil)
			return 0, nil, io.EOF
		case wsContinuation:
			msg = append(msg, data...)
		case wsText, wsBinary:
			msg, msgOp = data, op
		}
		if len(msg) > wsMaxMessage {
			return 0, nil, fmt.Errorf("websocket message too large")
		}
		if fin && op != wsPing && op != wsPong {
			return msgOp, msg, nil
		}
	}
}

// readFrame reads one frame. Per RFC 6455 every client→server frame is masked;
// we unmask in place.
func (c *wsConn) readFrame() (fin bool, opcode byte, payload []byte, err error) {
	var h [2]byte
	if _, err := io.ReadFull(c.br, h[:]); err != nil {
		return false, 0, nil, err
	}
	fin = h[0]&0x80 != 0
	opcode = h[0] & 0x0f
	masked := h[1]&0x80 != 0
	n := uint64(h[1] & 0x7f)
	switch n {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(c.br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		n = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(c.br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		n = binary.BigEndian.Uint64(ext[:])
	}
	if n > wsMaxMessage {
		return false, 0, nil, fmt.Errorf("websocket frame too large")
	}
	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(c.br, mask[:]); err != nil {
			return false, 0, nil, err
		}
	}
	payload = make([]byte, n)
	if _, err := io.ReadFull(c.br, payload); err != nil {
		return false, 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return fin, opcode, payload, nil
}

// WriteBinary sends a binary message (server→server frames are never masked).
func (c *wsConn) WriteBinary(p []byte) error { return c.write(wsBinary, p) }

func (c *wsConn) write(opcode byte, payload []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	var head [10]byte
	head[0] = 0x80 | opcode // FIN + opcode (we never fragment outbound)
	n := len(payload)
	switch {
	case n < 126:
		head[1] = byte(n)
		if _, err := c.conn.Write(head[:2]); err != nil {
			return err
		}
	case n < 1<<16:
		head[1] = 126
		binary.BigEndian.PutUint16(head[2:4], uint16(n))
		if _, err := c.conn.Write(head[:4]); err != nil {
			return err
		}
	default:
		head[1] = 127
		binary.BigEndian.PutUint64(head[2:10], uint64(n))
		if _, err := c.conn.Write(head[:10]); err != nil {
			return err
		}
	}
	if len(payload) > 0 {
		if _, err := c.conn.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

// Close sends a close frame (best-effort) and drops the connection.
func (c *wsConn) Close() error {
	_ = c.write(wsClose, nil)
	return c.conn.Close()
}
