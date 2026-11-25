package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("q") {
		case "results":
			fmt.Fprint(w, `{"results":[{"title":"a","url":"u","snippet":"s"}]}`)
		case "data":
			fmt.Fprint(w, `{"data":[{"title":"b"}]}`)
		case "bad":
			fmt.Fprint(w, `bukan json`)
		default:
			w.WriteHeader(500)
		}
	}))
	defer srv.Close()
	c := NewSearchClient(srv.URL, "key", srv.Client())

	res, err := c.Search(ctx, "results")
	require.NoError(t, err)
	assert.Equal(t, "a", res[0].Title)
	res, err = c.Search(ctx, "data")
	require.NoError(t, err)
	assert.Equal(t, "b", res[0].Title)

	_, err = c.Search(ctx, "bad")
	assert.Error(t, err)
	_, err = c.Search(ctx, "x")
	assert.Error(t, err)
	_, err = NewSearchClient("://bad", "k", srv.Client()).Search(ctx, "x")
	assert.Error(t, err)
	_, err = NewSearchClient("http://127.0.0.1:1", "k", srv.Client()).Search(ctx, "x")
	assert.Error(t, err)
}

func TestChat(t *testing.T) {
	mode := "ok"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch mode {
		case "ok":
			fmt.Fprint(w, `{"choices":[{"message":{"content":"halo"}}]}`)
		case "empty":
			fmt.Fprint(w, `{"choices":[]}`)
		case "bad":
			fmt.Fprint(w, `bukan json`)
		default:
			w.WriteHeader(500)
		}
	}))
	defer srv.Close()
	c := NewChatClient(srv.URL+"/", "key", "m", srv.Client())

	out, err := c.Chat(ctx, "s", "u")
	require.NoError(t, err)
	assert.Equal(t, "halo", out)

	for _, m := range []string{"empty", "bad", "error"} {
		mode = m
		_, err = c.Chat(ctx, "s", "u")
		assert.Error(t, err, m)
	}
	_, err = NewChatClient("://bad", "k", "m", srv.Client()).Chat(ctx, "s", "u")
	assert.Error(t, err)
	_, err = NewChatClient("http://127.0.0.1:1", "k", "m", srv.Client()).Chat(ctx, "s", "u")
	assert.Error(t, err)
}

type fakeSearch struct {
	res []SearchResult
	err error
}

func (f fakeSearch) Search(context.Context, string) ([]SearchResult, error) { return f.res, f.err }

type fakeChat struct {
	out string
	err error
}

func (f fakeChat) Chat(context.Context, string, string) (string, error) { return f.out, f.err }

const goodJSON = "```json\n{\"title\":\"T\",\"category\":\"teknologi\",\"excerpt\":\"e\",\"content\":\"c\"}\n```"

func TestGenerate(t *testing.T) {
	src := []SearchResult{{Title: "a", URL: "u", Snippet: "s"}}

	s := NewService(fakeSearch{res: src}, fakeChat{out: goodJSON})
	a, err := s.Generate(ctx, GenerateRequest{Topic: "go", IncludeWebSearch: true})
	require.NoError(t, err)
	assert.Equal(t, "Teknologi", a.Category)
	assert.Len(t, a.Sources, 1)

	s = NewService(fakeSearch{}, fakeChat{out: `{"title":"T","category":"ngawur"}`})
	a, err = s.Generate(ctx, GenerateRequest{Topic: "go"})
	require.NoError(t, err)
	assert.Equal(t, "Umum", a.Category)
	assert.Empty(t, a.Sources)

	_, err = NewService(fakeSearch{err: errors.New("x")}, fakeChat{out: goodJSON}).Generate(ctx, GenerateRequest{Topic: "go", IncludeWebSearch: true})
	assert.Error(t, err)
	_, err = NewService(fakeSearch{}, fakeChat{err: errors.New("x")}).Generate(ctx, GenerateRequest{Topic: "go"})
	assert.Error(t, err)
	_, err = NewService(fakeSearch{}, fakeChat{out: "tanpa json"}).Generate(ctx, GenerateRequest{Topic: "go"})
	assert.Error(t, err)
	_, err = NewService(fakeSearch{}, fakeChat{out: "{rusak}"}).Generate(ctx, GenerateRequest{Topic: "go"})
	assert.Error(t, err)
}

func TestHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	post := func(svc *Service, body string) int {
		r := gin.New()
		r.POST("/gen", NewHandler(svc).Generate)
		req := httptest.NewRequest("POST", "/gen", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	okSvc := NewService(fakeSearch{}, fakeChat{out: goodJSON})
	assert.Equal(t, 400, post(okSvc, `{}`))
	assert.Equal(t, 200, post(okSvc, `{"topic":"go"}`))
	assert.Equal(t, 502, post(NewService(fakeSearch{}, fakeChat{err: errors.New("x")}), `{"topic":"go"}`))
}
