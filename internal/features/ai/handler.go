package ai

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"app/internal/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data tidak valid: "+err.Error())
		return
	}
	article, err := h.svc.Generate(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Gagal menghasilkan artikel: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Draf artikel berhasil dibuat", article)
}
