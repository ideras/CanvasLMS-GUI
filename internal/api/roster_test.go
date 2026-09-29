package api_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"canvaslms-gui/internal/api"
	"github.com/stretchr/testify/require"
)

func TestGradeRosterPagination(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/courses/1/users", r.URL.Path)
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `[{"id":1002,"name":"Bob"}]`)
			return
		}
		require.Equal(t, "student", r.URL.Query().Get("enrollment_type[]"))
		w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses/1/users?page=2>; rel="next"`, server.URL))
		fmt.Fprint(w, `[{"id":1001,"name":"Alice"}]`)
	}))
	defer server.Close()
	client, err := api.NewCanvasClientWithToken(server.URL, "test-token")
	require.NoError(t, err)
	roster, err := client.GetStudentsForCourse(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, roster, 2)
	require.Equal(t, 1002, roster[1].ID)
}
