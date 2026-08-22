package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	managedRechargeControlBodyMaxBytes = 16 * 1024
	managedRechargeSessionBodyMaxBytes = 320 * 1024
	managedRechargeImportBodyMaxBytes  = 640 * 1024
)

type ManagedRechargeHandler struct {
	service *service.ManagedRechargeService
}

func NewManagedRechargeHandler(managedRechargeService *service.ManagedRechargeService) *ManagedRechargeHandler {
	return &ManagedRechargeHandler{service: managedRechargeService}
}

type managedRechargeCreateOrderRequest struct {
	ProductID   int64  `json:"product_id" binding:"required"`
	SessionJSON string `json:"session_json" binding:"required"`
}

type managedRechargeReplacementSessionRequest struct {
	SessionJSON string `json:"session_json" binding:"required"`
}

type managedRechargeImportRequest struct {
	ProductID int64    `json:"product_id" binding:"required"`
	Codes     []string `json:"codes" binding:"required"`
	ExpiresAt string   `json:"expires_at"`
}

type managedRechargeCDKStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type managedRechargeCDKProductRequest struct {
	ProductID int64 `json:"product_id" binding:"required"`
}

func (h *ManagedRechargeHandler) Catalog(c *gin.Context) {
	userID, ok := managedRechargeUserID(c)
	if !ok {
		return
	}
	result, err := h.service.GetCatalog(c.Request.Context(), userID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *ManagedRechargeHandler) CreateOrder(c *gin.Context) {
	userID, ok := managedRechargeUserID(c)
	if !ok {
		return
	}
	var req managedRechargeCreateOrderRequest
	if !managedRechargeBindJSON(c, &req, managedRechargeSessionBodyMaxBytes, "Invalid recharge request") {
		return
	}
	result, err := h.service.CreateOrder(c.Request.Context(), userID, service.ManagedRechargeCreateOrderInput{
		ProductID:      req.ProductID,
		Session:        req.SessionJSON,
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, managedRechargeUserOrder(result))
}

func (h *ManagedRechargeHandler) ListOrders(c *gin.Context) {
	userID, ok := managedRechargeUserID(c)
	if !ok {
		return
	}
	orders, err := h.service.ListUserOrders(c.Request.Context(), userID, managedRechargeLimit(c, 20))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, managedRechargeUserOrders(orders))
}

func (h *ManagedRechargeHandler) GetOrder(c *gin.Context) {
	userID, ok := managedRechargeUserID(c)
	if !ok {
		return
	}
	orderID, ok := managedRechargePathID(c)
	if !ok {
		return
	}
	order, err := h.service.GetOrder(c.Request.Context(), userID, orderID, false)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, managedRechargeUserOrder(order))
}

func (h *ManagedRechargeHandler) SubmitReplacementSession(c *gin.Context) {
	userID, ok := managedRechargeUserID(c)
	if !ok {
		return
	}
	orderID, ok := managedRechargePathID(c)
	if !ok {
		return
	}
	var req managedRechargeReplacementSessionRequest
	if !managedRechargeBindJSON(c, &req, managedRechargeSessionBodyMaxBytes, "Invalid Session request") {
		return
	}
	order, err := h.service.SubmitReplacementSession(c.Request.Context(), userID, orderID, req.SessionJSON)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, managedRechargeUserOrder(order))
}

func (h *ManagedRechargeHandler) AdminListProducts(c *gin.Context) {
	products, err := h.service.ListProducts(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, products)
}

func (h *ManagedRechargeHandler) AdminCreateProduct(c *gin.Context) {
	var req service.ManagedRechargeProductInput
	if !managedRechargeBindJSON(c, &req, managedRechargeControlBodyMaxBytes, "Invalid product request") {
		return
	}
	product, err := h.service.CreateProduct(c.Request.Context(), req)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, product)
}

func (h *ManagedRechargeHandler) AdminUpdateProduct(c *gin.Context) {
	productID, ok := managedRechargePathID(c)
	if !ok {
		return
	}
	var req service.ManagedRechargeProductInput
	if !managedRechargeBindJSON(c, &req, managedRechargeControlBodyMaxBytes, "Invalid product request") {
		return
	}
	product, err := h.service.UpdateProduct(c.Request.Context(), productID, req)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, product)
}

func (h *ManagedRechargeHandler) AdminImportCDKs(c *gin.Context) {
	adminID, ok := managedRechargeUserID(c)
	if !ok {
		return
	}
	var req managedRechargeImportRequest
	if !managedRechargeBindJSON(c, &req, managedRechargeImportBodyMaxBytes, "Invalid CDK import") {
		return
	}
	var expiresAt *time.Time
	if value := strings.TrimSpace(req.ExpiresAt); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			response.BadRequest(c, "Invalid CDK expiration time")
			return
		}
		expiresAt = &parsed
	}
	result, err := h.service.ImportCDKs(c.Request.Context(), adminID, req.ProductID, req.Codes, expiresAt)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, result)
}

func (h *ManagedRechargeHandler) AdminListCDKs(c *gin.Context) {
	productID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("product_id")), 10, 64)
	items, err := h.service.ListCDKs(c.Request.Context(), productID, c.Query("status"), managedRechargeLimit(c, 200))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, items)
}

func (h *ManagedRechargeHandler) AdminVerifyCDK(c *gin.Context) {
	cdkID, ok := managedRechargePathID(c)
	if !ok {
		return
	}
	result, err := h.service.VerifyCDK(c.Request.Context(), cdkID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *ManagedRechargeHandler) AdminSetCDKStatus(c *gin.Context) {
	cdkID, ok := managedRechargePathID(c)
	if !ok {
		return
	}
	var req managedRechargeCDKStatusRequest
	if !managedRechargeBindJSON(c, &req, managedRechargeControlBodyMaxBytes, "Invalid CDK status request") {
		return
	}
	if err := h.service.SetCDKStatus(c.Request.Context(), cdkID, req.Status); response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{"status": req.Status})
}

func (h *ManagedRechargeHandler) AdminMoveCDK(c *gin.Context) {
	cdkID, ok := managedRechargePathID(c)
	if !ok {
		return
	}
	var req managedRechargeCDKProductRequest
	if !managedRechargeBindJSON(c, &req, managedRechargeControlBodyMaxBytes, "Invalid CDK product request") {
		return
	}
	if err := h.service.MoveCDK(c.Request.Context(), cdkID, req.ProductID); response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{"product_id": req.ProductID})
}

func (h *ManagedRechargeHandler) AdminListOrders(c *gin.Context) {
	orders, err := h.service.ListAdminOrders(c.Request.Context(), c.Query("status"), managedRechargeLimit(c, 100))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, orders)
}

func (h *ManagedRechargeHandler) AdminSyncOrder(c *gin.Context) {
	orderID, ok := managedRechargePathID(c)
	if !ok {
		return
	}
	order, err := h.service.AdminGetOrder(c.Request.Context(), orderID, true)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, order)
}

func (h *ManagedRechargeHandler) AdminRefundOrder(c *gin.Context) {
	adminID, ok := managedRechargeUserID(c)
	if !ok {
		return
	}
	orderID, ok := managedRechargePathID(c)
	if !ok {
		return
	}
	order, err := h.service.AdminRefundOrder(c.Request.Context(), adminID, orderID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, order)
}

func managedRechargeUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

func managedRechargePathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid resource ID")
		return 0, false
	}
	return id, true
}

func managedRechargeLimit(c *gin.Context, fallback int) int {
	limit, err := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	if err != nil || limit <= 0 {
		return fallback
	}
	return limit
}

func managedRechargeBindJSON(c *gin.Context, dst any, maxBytes int64, invalidMessage string) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
	if err := c.ShouldBindJSON(dst); err != nil {
		if _, ok := extractMaxBytesError(err); ok {
			response.Error(c, http.StatusRequestEntityTooLarge, "Recharge request is too large")
		} else {
			response.BadRequest(c, invalidMessage)
		}
		return false
	}
	return true
}

func managedRechargeUserOrders(orders []service.ManagedRechargeOrder) []service.ManagedRechargeOrder {
	result := make([]service.ManagedRechargeOrder, len(orders))
	for i := range orders {
		result[i] = *managedRechargeUserOrder(&orders[i])
	}
	return result
}

func managedRechargeUserOrder(order *service.ManagedRechargeOrder) *service.ManagedRechargeOrder {
	if order == nil {
		return nil
	}
	result := *order
	result.CDKMasked = ""
	result.UpstreamStatus = ""
	result.Progress = ""
	result.ErrorCode = ""
	result.UserEmail = ""
	result.Username = ""
	return &result
}
