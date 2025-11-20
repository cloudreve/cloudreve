package h3

import (
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-reuseport"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

type H3Server struct {
	*http3.Server
	localAddr string
	pubAddr   string
	conn      net.PacketConn

	incoming chan string
}

func NewH3Server() (*H3Server, error) {
	localAddr := "0.0.0.0:0"
	localAddr, pubLocalAddr, err := GetPublicAddrWithFallback(localAddr)
	if err != nil {
		return nil, err
	}

	conn, err := reuseport.ListenPacket("udp4", localAddr)
	if err != nil {
		return nil, err
	}

	tlsConfig := GenerateTLSConfig()
	quicConfig := &quic.Config{
		KeepAlivePeriod: 15 * time.Second,
		EnableDatagrams: true,
		MaxIdleTimeout:  time.Hour,
	}

	server := &http3.Server{
		Addr:        conn.LocalAddr().String(),
		QUICConfig:  quicConfig,
		TLSConfig:   tlsConfig,
		IdleTimeout: time.Hour,
	}

	return &H3Server{
		localAddr: localAddr,
		pubAddr:   pubLocalAddr,
		conn:      conn,
		Server:    server,
		incoming:  make(chan string, 1),
	}, nil
}

func (s *H3Server) GetAddrs() (string, string) {
	return s.localAddr, s.pubAddr
}

func (s *H3Server) Serve() error {
	return s.Server.Serve(s.conn)
}
func (s *H3Server) Close() error {
	var errs = make([]error, 0, 2)
	if s.Server != nil {
		if err := s.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	case 2:
		return fmt.Errorf("close server err: %v, close conn err: %v", errs[0], errs[1])
	}
	return nil
}

func (s *H3Server) HandleRemote(remoteAddr string) error {
	return PunchHole(s.localAddr, remoteAddr)
}
