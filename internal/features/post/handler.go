package post

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"app/internal/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func handleErr(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		response.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
}

func respond(c *gin.Context, code int, msg string, data interface{}, err error) {
	if err != nil {
		handleErr(c, err)
		return
	}
	response.Success(c, code, msg, data)
}

func bind(c *gin.Context, req *PostRequest) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data tidak valid: "+err.Error())
		return false
	}
	return true
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return uint(id), true
}

func (h *Handler) Categories(c *gin.Context) {
	response.Success(c, http.StatusOK, "Daftar kategori", Categories)
}

func (h *Handler) ListPublic(c *gin.Context) {
	posts, err := h.svc.ListPublic(c.Query("q"), c.Query("category"))
	respond(c, http.StatusOK, "Daftar artikel", posts, err)
}

func (h *Handler) GetPublic(c *gin.Context) {
	p, err := h.svc.GetPublic(c.Param("slug"))
	respond(c, http.StatusOK, "Detail artikel", p, err)
}

func (h *Handler) ListMine(c *gin.Context) {
	posts, err := h.svc.ListMine(c.GetUint("user_id"))
	respond(c, http.StatusOK, "Daftar artikel saya", posts, err)
}

func (h *Handler) Stats(c *gin.Context) {
	s, err := h.svc.Stats(c.GetUint("user_id"))
	respond(c, http.StatusOK, "Statistik artikel", s, err)
}

func (h *Handler) GetMine(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.GetMine(id, c.GetUint("user_id"))
	respond(c, http.StatusOK, "Detail artikel", p, err)
}

func (h *Handler) Create(c *gin.Context) {
	var req PostRequest
	if !bind(c, &req) {
		return
	}
	p, err := h.svc.Create(c.GetUint("user_id"), req)
	respond(c, http.StatusCreated, "Artikel dibuat", p, err)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req PostRequest
	if !bind(c, &req) {
		return
	}
	p, err := h.svc.Update(id, c.GetUint("user_id"), req)
	respond(c, http.StatusOK, "Artikel diperbarui", p, err)
}

func (h *Handler) Toggle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.Toggle(id, c.GetUint("user_id"))
	respond(c, http.StatusOK, "Status publikasi diubah", p, err)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	err := h.svc.Delete(id, c.GetUint("user_id"))
	respond(c, http.StatusOK, "Artikel dihapus", nil, err)
}

func (h *Handler) AdminUsers(c *gin.Context) {
	list, err := h.svc.AdminUsers()
	respond(c, http.StatusOK, "Daftar pengguna", list, err)
}

func (h *Handler) AdminPosts(c *gin.Context) {
	posts, err := h.svc.AdminPosts()
	respond(c, http.StatusOK, "Arsip seluruh artikel", posts, err)
}

func (h *Handler) AdminUserPosts(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	posts, err := h.svc.ListMine(id)
	respond(c, http.StatusOK, "Artikel pengguna", posts, err)
}
