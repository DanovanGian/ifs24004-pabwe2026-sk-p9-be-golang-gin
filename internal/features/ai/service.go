package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"app/internal/features/post"
)

type Searcher interface {
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

type Chatter interface {
	Chat(ctx context.Context, system, user string) (string, error)
}

const systemPrompt = `Kamu adalah penulis blog profesional berbahasa Indonesia. ` +
	`Balas HANYA dengan JSON valid berformat {"title": string, "category": string, "excerpt": string, "content": string}. ` +
	`Field content berisi artikel lengkap dalam format Markdown.`

type Service struct {
	search Searcher
	chat   Chatter
}

func NewService(s Searcher, c Chatter) *Service { return &Service{search: s, chat: c} }

func (s *Service) Generate(ctx context.Context, req GenerateRequest) (*Article, error) {
	sources := []SearchResult{}
	if req.IncludeWebSearch {
		found, err := s.search.Search(ctx, req.Topic)
		if err != nil {
			return nil, err
		}
		sources = found
	}
	raw, err := s.chat.Chat(ctx, systemPrompt, buildPrompt(req, sources))
	if err != nil {
		return nil, err
	}
	article, err := parseArticle(raw)
	if err != nil {
		return nil, err
	}
	article.Category = normalizeCategory(article.Category)
	article.Sources = sources
	return article, nil
}

func buildPrompt(req GenerateRequest, sources []SearchResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Tulis artikel blog tentang: %s\nKategori: %s (pilihan: %s)\n",
		req.Topic, req.Category, strings.Join(post.Categories, ", "))
	if len(sources) > 0 {
		b.WriteString("\nGunakan fakta terkini dari hasil penelusuran berikut dan sebutkan sumbernya:\n")
		for i, s := range sources {
			fmt.Fprintf(&b, "%d. %s (%s): %s\n", i+1, s.Title, s.URL, s.Snippet)
		}
	}
	return b.String()
}

func parseArticle(raw string) (*Article, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return nil, errors.New("respons AI tidak berisi JSON")
	}
	var a Article
	if err := json.Unmarshal([]byte(raw[start:end+1]), &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func normalizeCategory(c string) string {
	for _, v := range post.Categories {
		if strings.EqualFold(v, c) {
			return v
		}
	}
	return "Umum"
}
