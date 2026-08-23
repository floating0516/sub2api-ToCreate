package handler

import (
	"encoding/json"
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
	SessionJSON string `json:"session_json"`
	ReturnURL   string `json:"return_url"`
	IsMobile    *bool  `json:"is_mobile,omitempty"`
}

type managedRechargeValidateSessionRequest struct {
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

func (h *ManagedRechargeHandler) ValidateSession(c *gin.Context) {
	if _, ok := managedRechargeUserID(c); !ok {
		return
	}
	var req managedRechargeValidateSessionRequest
	if !managedRechargeBindJSON(c, &req, managedRechargeSessionBodyMaxBytes, "Invalid Session request") {
		return
	}
	result, err := h.service.ValidateSession(c.Request.Context(), req.SessionJSON)
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
	mobile := isMobile(c)
	if req.IsMobile != nil {
		mobile = *req.IsMobile
	}
	result, err := h.service.CreateCheckout(c.Request.Context(), userID, service.ManagedRechargeCreateOrderInput{
		ProductID:      req.ProductID,
		Session:        req.SessionJSON,
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		ClientIP:       c.ClientIP(),
		IsMobile:       mobile,
		SrcHost:        c.Request.Host,
		SrcURL:         c.Request.Referer(),
		ReturnURL:      req.ReturnURL,
		Locale:         c.GetHeader("Accept-Language"),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, managedRechargeUserCheckout(result))
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

func (h *ManagedRechargeHandler) GetOrderStatus(c *gin.Context) {
	userID, ok := managedRechargeUserID(c)
	if !ok {
		return
	}
	orderID, ok := managedRechargePathID(c)
	if !ok {
		return
	}
	order, err := h.service.GetOrderStatus(c.Request.Context(), userID, orderID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, managedRechargeUserOrderStatus(order))
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

func managedRechargeUserCheckout(checkout *service.ManagedRechargeCheckout) *service.ManagedRechargeCheckout {
	if checkout == nil {
		return nil
	}
	result := *checkout
	result.Order = managedRechargeUserOrder(checkout.Order)
	return &result
}

func managedRechargeUserOrder(order *service.ManagedRechargeOrder) *service.ManagedRechargeOrder {
	if order == nil {
		return nil
	}
	result := *order
	result.CDKMasked = ""
	result.UpstreamStatus = ""
	result.Progress = managedRechargeUserProgress(order)
	result.ErrorCode = ""
	result.UserEmail = ""
	result.Username = ""
	return &result
}

func managedRechargeUserOrderStatus(order *service.ManagedRechargeOrder) *service.ManagedRechargeOrder {
	result := managedRechargeUserOrder(order)
	if result == nil {
		return nil
	}
	result.RedemptionCode = ""
	result.RedemptionURL = ""
	return result
}

func managedRechargeUserProgress(order *service.ManagedRechargeOrder) string {
	if order == nil {
		return ""
	}
	switch order.Status {
	case service.ManagedRechargeStatusAwaitingPayment:
		return "等待支付宝付款"
	case service.ManagedRechargeStatusValidating:
		return "正在确认库存与订单"
	case service.ManagedRechargeStatusPaid, service.ManagedRechargeStatusSubmitting:
		return "支付成功，正在提交兑换任务"
	case service.ManagedRechargeStatusIssued:
		return "CDK 已发放，等待前往兑换页提交"
	case service.ManagedRechargeStatusQueued:
		return "兑换任务已进入处理队列"
	case service.ManagedRechargeStatusProcessing:
		return managedRechargeProcessingSummary(order.Progress)
	case service.ManagedRechargeStatusVerifying:
		return "付款已完成，正在确认订阅状态"
	case service.ManagedRechargeStatusActionRequired:
		return "需要前往兑换页补交新的 Session"
	case service.ManagedRechargeStatusManualReview:
		return "订单正在人工核对"
	case service.ManagedRechargeStatusCompleted:
		return "订阅已确认完成"
	case service.ManagedRechargeStatusFailed:
		return "订单未完成"
	case service.ManagedRechargeStatusRefunded:
		return "订单未完成，款项已退回"
	default:
		return ""
	}
}

func managedRechargeProcessingSummary(raw string) string {
	var progress struct {
		Step string `json:"step"`
	}
	if json.Unmarshal([]byte(raw), &progress) == nil {
		switch strings.ToLower(strings.TrimSpace(progress.Step)) {
		case "login", "login_email", "login_otp", "navigate":
			return "正在验证目标账号"
		case "checkout", "filling":
			return "正在创建订阅订单"
		case "submitting", "3ds_pending":
			return "正在提交付款"
		case "verifying", "upgrading", "paid", "canceling", "cancel_retry":
			return "付款已完成，正在确认订阅状态"
		case "completed":
			return "订阅已确认完成"
		}
	}
	return "兑换服务正在处理"
}
