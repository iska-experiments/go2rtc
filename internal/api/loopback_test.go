package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIIsLoopbackOnly(t *testing.T) {
	HandleFunc("api/loopback-test", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	HandleFunc("/pair-test", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })

	for _, tc := range []struct {
		path, remote string
		want         int
	}{
		{"/api/loopback-test", "127.0.0.1:1234", http.StatusNoContent},
		{"/api/loopback-test", "192.168.88.29:1234", http.StatusForbidden},
		{"/pair-test", "192.168.88.29:1234", http.StatusNoContent},
	} {
		req := httptest.NewRequest("GET", tc.path, nil)
		req.RemoteAddr = tc.remote
		rec := httptest.NewRecorder()
		http.DefaultServeMux.ServeHTTP(rec, req)
		require.Equal(t, tc.want, rec.Code, tc.path+" from "+tc.remote)
	}
}
