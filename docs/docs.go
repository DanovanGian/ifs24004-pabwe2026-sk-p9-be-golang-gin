package docs

import (
	"encoding/json"
	"regexp"
	"strings"
)

type Route struct {
	Method, Path, Tag, Summary string
	Auth                       bool
	Query                      []string
}

func rt(method, path, tag, summary string, auth bool, query ...string) Route {
	return Route{Method: method, Path: path, Tag: tag, Summary: summary, Auth: auth, Query: query}
}

var Routes = []Route{
	rt("GET", "/api/public/categories", "Public", "Daftar kategori artikel", false),
	rt("GET", "/api/public/posts", "Public", "Daftar artikel terbit (q, category)", false, "q", "category"),
	rt("GET", "/api/public/posts/:slug", "Public", "Detail artikel berdasarkan slug", false),
	rt("POST", "/api/auth/register", "Auth", "Registrasi penulis baru", false),
	rt("POST", "/api/auth/login", "Auth", "Login dan terbitkan token JWT", false),
	rt("POST", "/api/auth/logout", "Auth", "Logout", true),
	rt("GET", "/api/auth/profile", "Auth", "Ambil profil", true),
	rt("PUT", "/api/auth/profile", "Auth", "Perbarui profil", true),
	rt("GET", "/api/posts", "Author", "Daftar artikel milik penulis", true),
	rt("POST", "/api/posts", "Author", "Buat artikel baru", true),
	rt("GET", "/api/posts/stats", "Author", "Statistik artikel penulis", true),
	rt("POST", "/api/posts/ai-generate", "Author", "Generate draf artikel dengan AI + Search Engine", true),
	rt("GET", "/api/posts/:id", "Author", "Detail artikel milik penulis", true),
	rt("PUT", "/api/posts/:id", "Author", "Perbarui artikel", true),
	rt("PATCH", "/api/posts/:id/toggle-publish", "Author", "Ubah status draf/terbit", true),
	rt("DELETE", "/api/posts/:id", "Author", "Hapus artikel", true),
	rt("GET", "/api/admin/users", "Admin", "Daftar seluruh pengguna", true),
	rt("GET", "/api/admin/posts", "Admin", "Arsip seluruh artikel", true),
	rt("GET", "/api/admin/users/:id/posts", "Admin", "Artikel milik pengguna tertentu", true),
}

var paramRe = regexp.MustCompile(`:([A-Za-z_]+)`)

func parameters(r Route) []map[string]interface{} {
	var params []map[string]interface{}
	for _, m := range paramRe.FindAllStringSubmatch(r.Path, -1) {
		params = append(params, map[string]interface{}{
			"name": m[1], "in": "path", "required": true, "schema": map[string]string{"type": "string"},
		})
	}
	for _, q := range r.Query {
		params = append(params, map[string]interface{}{
			"name": q, "in": "query", "schema": map[string]string{"type": "string"},
		})
	}
	return params
}

func GetSwaggerJSON(host, scheme string) []byte {
	paths := map[string]map[string]interface{}{}
	for _, r := range Routes {
		p := paramRe.ReplaceAllString(r.Path, "{$1}")
		if paths[p] == nil {
			paths[p] = map[string]interface{}{}
		}
		op := map[string]interface{}{
			"tags":      []string{r.Tag},
			"summary":   r.Summary,
			"responses": map[string]interface{}{"200": map[string]string{"description": "OK"}},
		}
		if params := parameters(r); len(params) > 0 {
			op["parameters"] = params
		}
		if r.Auth {
			op["security"] = []map[string][]string{{"bearerAuth": {}}}
		}
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			op["requestBody"] = map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{"schema": map[string]string{"type": "object"}},
				},
			}
		}
		paths[p][strings.ToLower(r.Method)] = op
	}
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info":    map[string]string{"title": "Delcom Blog API", "version": "1.0.0", "description": "REST API Delcom Blog (Golang Gin)"},
		"servers": []map[string]string{{"url": scheme + "://" + host}},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]string{"type": "http", "scheme": "bearer", "bearerFormat": "JWT"},
			},
		},
		"paths": paths,
	}
	out, _ := json.Marshal(spec)
	return out
}
