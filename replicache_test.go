package replicache

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MainSuite struct {
	suite.Suite
}

func TestMainSuite(t *testing.T) {
	suite.Run(t, new(MainSuite))
}

type MockAuth func(ctx context.Context, token string) bool

type Todo struct {
	ID   int
	Text string
}

func (suite *MainSuite) TestNewReplicacheInstance() {
	authCalled := 0
	authFn := func(ctx context.Context, token string) bool {
		authCalled++
		return true
	}

	r := New[Todo](WithAuth(authFn))
	r.Register("todo", func(m Mutation) {
		// m.Name
	})
}

func (s *MainSuite) TestRequestWithAuth() {
	ctx := context.TODO()
	authCalled := 0
	authFn := func(ctx context.Context, token string) bool {
		authCalled++
		return token == "TOKEN"
	}

	r := New[Todo](WithAuth(authFn))
	handler := r.HandlePull(func(pr *PullRequest, spaceID string) (PullResponse[Todo], error) {
		return PullResponse[Todo]{}, nil
	})

	body := bytes.NewBuffer([]byte(`{"token":"test"}`))

	buf := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, "POST", DefaultPullEndpoint, body)
	s.NoError(err)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(ReplicacheRequestIDHeader, "1")
	req.Header.Add(authorizationHeader, "TOKEN")

	handler(buf, req)

	s.Equal(1, authCalled)
	s.Equal(200, buf.Result().StatusCode)
}
