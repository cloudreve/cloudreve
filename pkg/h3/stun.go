package h3

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/libp2p/go-reuseport"
	"github.com/pion/stun/v3"
)

type stunAddr struct {
	localAddr string
	pubAddr   string
}

// 尝试多个STUN服务器
func GetPublicAddrWithFallback(localAddr string) (string, string, error) {
	servers := []string{
		"stun.miwifi.com:3478",
		"stun.chat.bilibili.com:3478",
		"turn.cloudflare.com:3478",
		"fwa.lifesizecloud.com:3478",
		"stun.isp.net.au:3478",
		"stun.voipbusterpro.com:3478",
		"stun.freeswitch.org:3478",
		"stun.nextcloud.com:3478",
		"stun.l.google.com:19302",
		"stun.sipnet.com:3478",
	}
	ctx, cancel := context.WithCancel(context.Background())

	var ch = make(chan stunAddr, 1)
	defer func() {
		cancel()
		close(ch)
	}()
	var wg sync.WaitGroup
	for _, server := range servers {
		wg.Add(1)
		go func(server string) {
			defer func() {
				recover()
				wg.Done()
			}()

			localAddr, pubAddr, err := getPublicAddr(ctx, localAddr, server)
			if err == nil {
				select {
				case <-ctx.Done():
					return
				default:
					ch <- stunAddr{
						localAddr: localAddr,
						pubAddr:   pubAddr,
					}
				}
				return
			}
		}(server)

	}
	go func() {
		wg.Wait()
		cancel()
	}()
	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case addr, ok := <-ch:
		if !ok {
			return "", "", errors.New("all STUN servers failed")
		}
		return addr.localAddr, addr.pubAddr, nil
	}
}

type stunConn struct {
	net.PacketConn
	raddr *net.UDPAddr
}

func (c *stunConn) Write(data []byte) (int, error) {
	return c.WriteTo(data, c.raddr)
}
func (c *stunConn) Read(data []byte) (int, error) {
	n, _, err := c.PacketConn.ReadFrom(data)
	return n, err
}

func getPublicAddr(ctx context.Context, laddr, stunServer string) (localAddr, pubAddr string, err error) {
	if laddr == "" {
		laddr = "0.0.0.0:0"
	}
	raddr, err := net.ResolveUDPAddr("udp", stunServer)
	if err != nil {
		return
	}
	conn, err := reuseport.ListenPacket("udp4", laddr)
	if err != nil {
		return
	}
	defer conn.Close()
	client, err := stun.NewClient(&stunConn{
		PacketConn: conn,
		raddr:      raddr,
	})
	if err != nil {
		return
	}
	defer client.Close()

	localAddr = conn.LocalAddr().String()
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	default:
		err = client.Do(message, func(res stun.Event) {
			if res.Error != nil {
				err = res.Error
				return
			}
			// Decoding XOR-MAPPED-ADDRESS attribute from message.
			var xorAddr stun.XORMappedAddress
			if err = xorAddr.GetFrom(res.Message); err != nil {
				return
			}
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return
			default:
				pubAddr = xorAddr.String()
			}
		})
		if err != nil {
			return "", "", err
		}
	}
	if err != nil {
		return
	}
	if pubAddr == "" {
		return "", "", errors.New("get pun addr fail")
	}

	return
}
