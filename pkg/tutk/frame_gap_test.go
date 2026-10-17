package tutk

import (
	"encoding/binary"
	"fmt"
	"testing"
)

// frame feeds one video frame of n packets; skip, if >= 0, is a packet index that never arrives.
func frame(h *FrameHandler, no uint32, n uint16, key bool, skip int) {
	flags := uint8(0)
	if key {
		flags = 0x01
	}
	// 400 ms apart, wrapping each second like the cameras' microsecond clock.
	fi := &FrameInfo{CodecID: CodecH264, Flags: flags, FrameNo: no, PayloadSize: uint32(n), Timestamp: no * 400000 % tsWrapPeriod}
	for i := uint16(0); i < n; i++ {
		if int(i) == skip {
			continue
		}
		var info *FrameInfo
		if i == n-1 {
			info = fi
		}
		h.handleVideo(ChannelIVideo, &PacketHeader{FrameNo: no, PktIdx: i, PktTotal: n}, []byte{byte(i)}, info)
	}
}

func received(h *FrameHandler) (frames []uint32) {
	for {
		select {
		case pkt := <-h.output:
			// RTP time (90 kHz) since frame 1 must match the frame's real time, gaps included.
			if want := (pkt.FrameNo - 1) * 400 * 90; pkt.Timestamp != want {
				panic(fmt.Sprintf("frame %d at %d, want %d", pkt.FrameNo, pkt.Timestamp, want))
			}
			frames = append(frames, pkt.FrameNo)
		default:
			return
		}
	}
}

func TestVideoWaitsForKeyframeAfterLoss(t *testing.T) {
	h := NewFrameHandler(false)
	frame(h, 1, 2, false, -1) // before the first keyframe: dropped
	frame(h, 2, 4, true, -1)
	frame(h, 3, 2, false, -1)
	frame(h, 4, 4, true, 1) // keyframe loses a packet
	frame(h, 5, 2, false, -1)
	frame(h, 6, 2, false, -1)
	frame(h, 7, 4, true, -1)
	frame(h, 8, 2, false, -1)
	got := received(h)
	want := []uint32{2, 3, 7, 8}
	if len(got) != len(want) {
		t.Fatalf("frames %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("frames %v, want %v", got, want)
		}
	}
}

func TestFrameInfoMarkerOnStartAltPacket(t *testing.T) {
	// The last of 5 packets, sent as StartAlt (36-byte header) with the 0x28 FRAMEINFO marker,
	// as Wyze Cam Pan v3 firmware does (seen 2026-10-16).
	data := make([]byte, 64)
	data[0], data[1] = ChannelPVideo, FrameTypeStartAlt
	binary.LittleEndian.PutUint16(data[20:], 5)
	binary.LittleEndian.PutUint16(data[22:], 0x0028)
	binary.LittleEndian.PutUint32(data[32:], 606311)
	hdr := ParsePacketHeader(data)
	if !hdr.HasFrameInfo || hdr.PktIdx != 4 {
		t.Fatalf("got idx %d frameinfo %v, want 4 true", hdr.PktIdx, hdr.HasFrameInfo)
	}
	// In a frame of more than 40 packets, 40 is a real index.
	binary.LittleEndian.PutUint16(data[20:], 120)
	hdr = ParsePacketHeader(data)
	if hdr.HasFrameInfo || hdr.PktIdx != 40 {
		t.Fatalf("got idx %d frameinfo %v, want 40 false", hdr.PktIdx, hdr.HasFrameInfo)
	}
}
