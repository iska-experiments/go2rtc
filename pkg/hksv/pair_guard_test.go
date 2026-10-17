package hksv

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/AlexxIT/go2rtc/pkg/hap/tlv8"
	"github.com/stretchr/testify/require"
)

func TestPairSetupRefusal(t *testing.T) {
	s := &Server{}
	require.Equal(t, byte(0), s.pairSetupRefusal())

	s.failedPairSetups.Store(pairSetupMaxTries)
	require.Equal(t, byte(tlvErrorMaxTries), s.pairSetupRefusal())

	s.failedPairSetups.Store(0)
	s.pairings = []string{"client_id=x&client_public=00&permissions=1"}
	require.Equal(t, byte(tlvErrorUnavailable), s.pairSetupRefusal())
}

func TestWritePairSetupError(t *testing.T) {
	var out bytes.Buffer
	rw := bufio.NewReadWriter(bufio.NewReader(&bytes.Buffer{}), bufio.NewWriter(&out))
	require.NoError(t, writePairSetupError(rw, tlvErrorUnavailable))

	_, body, ok := bytes.Cut(out.Bytes(), []byte("\r\n\r\n"))
	require.True(t, ok)
	var m2 struct {
		State byte `tlv8:"6"`
		Error byte `tlv8:"7"`
	}
	require.NoError(t, tlv8.Unmarshal(body, &m2))
	require.Equal(t, byte(2), m2.State)
	require.Equal(t, byte(tlvErrorUnavailable), m2.Error)
}
