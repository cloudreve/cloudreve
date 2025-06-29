package h3

import (
	"net"

	"github.com/libp2p/go-reuseport"
)

func PunchHole(localAddr, remoteAddr string) error {
	conn, err := reuseport.ListenPacket("udp4", localAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	remoteUDPAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return err
	}

	// tk := time.NewTicker(5 * time.Second)
	// defer tk.Stop()

	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()

	// var ch = make(chan bool)
	// go func() {
	// 	defer cancel()
	// 	defer func() {
	// 		cancel()
	// 		close(ch)
	// 	}()
	// 	buf := make([]byte, 512)
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 		default:
	// 			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	// 			_, raddr, err := conn.ReadFrom(buf)
	// 			if err != nil {
	// 				conn.SetReadDeadline(time.Time{}) // 清除超时
	// 				continue
	// 			}
	// 			if raddr.String() == remoteAddr {
	// 				ch <- true
	// 				return
	// 			}
	// 			conn.SetReadDeadline(time.Time{}) // 清除超时
	// 		}
	// 	}
	// }()

	conn.WriteTo([]byte("PUNCH"), remoteUDPAddr)
	return err
}
