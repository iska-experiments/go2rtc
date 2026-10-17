package dtls

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/tutk"
)

// A camera that stops sending frames must fail the read once the caller's deadline passes, so
// the producer exits and go2rtc reconnects. The reader goroutine's 100 ms socket deadlines used
// to override it, and the read then waited on the frame channel forever.
func TestAVRecvFrameDataHonoursDeadline(t *testing.T) {
	udp, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &DTLSConn{conn: udp, frames: tutk.NewFrameHandler(false), ctx: ctx, cancel: cancel}
	// What reader() does on every loop.
	_ = udp.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	if err = c.SetDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	_ = udp.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	done := make(chan error, 1)
	go func() {
		_, err := c.AVRecvFrameData()
		done <- err
	}()
	select {
	case err = <-done:
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("got %v, want os.ErrDeadlineExceeded", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("AVRecvFrameData ignored the deadline")
	}

	// A zero deadline waits for frames again.
	_ = c.SetDeadline(time.Time{})
	go func() {
		_, err := c.AVRecvFrameData()
		done <- err
	}()
	select {
	case err = <-done:
		t.Fatalf("returned %v without a deadline", err)
	case <-time.After(200 * time.Millisecond):
	}
}
