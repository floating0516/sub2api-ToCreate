package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type supportCreateRequest struct {
	Type            string      `json:"type"`
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	Priority        string      `json:"priority"`
	Product         string      `json:"product"`
	OrderID         *string     `json:"order_id,omitempty"`
	AttachmentIDs   []uuid.UUID `json:"attachment_ids,omitempty"`
	ClientRequestID uuid.UUID   `json:"client_request_id"`
}

type supportReplyRequest struct {
	Content         string      `json:"content"`
	AttachmentIDs   []uuid.UUID `json:"attachment_ids,omitempty"`
	ClientMessageID uuid.UUID   `json:"client_message_id"`
}

func (h *SupportHandler) BindOrderValidator(validator func(context.Context, int64, int64) error) {
	h.validateOrder = validator
}

func (h *SupportHandler) ListTickets(c *gin.Context) { h.listTickets(c, "/tickets") }
func (h *SupportHandler) TicketStats(c *gin.Context) {
	h.proxyJSON(c, http.MethodGet, "/tickets/stats", false)
}
func (h *SupportHandler) CreateTicket(c *gin.Context) {
	h.proxyJSON(c, http.MethodPost, "/tickets", true)
}
func (h *SupportHandler) ReplyTicket(c *gin.Context)      { h.ticketAction(c, "", "replies") }
func (h *SupportHandler) ResolveTicket(c *gin.Context)    { h.ticketAction(c, "", "resolve") }
func (h *SupportHandler) CloseTicket(c *gin.Context)      { h.ticketAction(c, "", "close") }
func (h *SupportHandler) ReopenTicket(c *gin.Context)     { h.ticketAction(c, "", "reopen") }
func (h *SupportHandler) AdminListTickets(c *gin.Context) { h.listTickets(c, "/admin/tickets") }
func (h *SupportHandler) AdminTicketStats(c *gin.Context) {
	h.proxyJSON(c, http.MethodGet, "/admin/tickets/stats", false)
}
func (h *SupportHandler) AdminGetTicket(c *gin.Context) {
	if h.validPortalID(c, "/admin/tickets") {
		h.proxyJSON(c, http.MethodGet, "/admin/tickets/"+c.Param("id"), false)
	}
}
func (h *SupportHandler) AdminReplyTicket(c *gin.Context)   { h.ticketAction(c, "/admin", "replies") }
func (h *SupportHandler) AdminResolveTicket(c *gin.Context) { h.ticketAction(c, "/admin", "resolve") }
func (h *SupportHandler) AdminCloseTicket(c *gin.Context)   { h.ticketAction(c, "/admin", "close") }
func (h *SupportHandler) AdminReopenTicket(c *gin.Context)  { h.ticketAction(c, "/admin", "reopen") }

func (h *SupportHandler) validPortalID(c *gin.Context, path string) bool {
	if !h.supportReady(c, path) {
		return false
	}
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		response.BadRequest(c, "invalid support id")
		return false
	}
	return true
}

func (h *SupportHandler) ticketAction(c *gin.Context, prefix, action string) {
	path := prefix + "/tickets"
	if h.validPortalID(c, path) {
		h.proxyJSON(c, http.MethodPost, path+"/"+c.Param("id")+"/"+action, action == "replies")
	}
}

func (h *SupportHandler) listTickets(c *gin.Context, path string) {
	if !h.supportReady(c, path) {
		return
	}
	query := url.Values{}
	for _, key := range []string{"q", "type", "status", "page", "page_size"} {
		if value, exists := c.GetQuery(key); exists {
			query.Set(key, value)
		}
	}
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	h.proxyJSON(c, http.MethodGet, path, false)
}

func (h *SupportHandler) readPortalRequest(c *gin.Context, path string) ([]byte, error) {
	if path == "/chat" || path == "/tickets/confirm" {
		return readSupportRequest(c)
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, supportMaxRequestBytes)
	var fields map[string]json.RawMessage
	dec := json.NewDecoder(c.Request.Body)
	if err := dec.Decode(&fields); err != nil {
		return nil, err
	}
	if fields == nil || dec.Decode(&struct{}{}) != io.EOF {
		return nil, errors.New("request body must contain one JSON object")
	}
	delete(fields, "user_id")
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	var payload any
	if path == "/tickets" {
		payload = &supportCreateRequest{}
	} else {
		payload = &supportReplyRequest{}
	}
	strict := json.NewDecoder(strings.NewReader(string(raw)))
	strict.DisallowUnknownFields()
	if err := strict.Decode(payload); err != nil {
		return nil, err
	}
	switch req := payload.(type) {
	case *supportCreateRequest:
		req.Title, req.Description, req.Product = strings.TrimSpace(req.Title), strings.TrimSpace(req.Description), strings.TrimSpace(req.Product)
		if !validSupportType(req.Type) || utf8.RuneCountInString(req.Title) < 4 || utf8.RuneCountInString(req.Title) > 200 || utf8.RuneCountInString(req.Description) < 10 || utf8.RuneCountInString(req.Description) > 8000 || req.Product == "" || utf8.RuneCountInString(req.Product) > 128 || req.ClientRequestID == uuid.Nil || !validSupportPriority(req.Priority) || !validSupportAttachments(req.AttachmentIDs) {
			return nil, errors.New("invalid ticket fields")
		}
		if req.Type == "order" && (req.OrderID == nil || strings.TrimSpace(*req.OrderID) == "") {
			return nil, errors.New("order_id is required for order tickets")
		}
		if req.OrderID != nil {
			*req.OrderID = strings.TrimSpace(*req.OrderID)
			orderID, parseErr := strconv.ParseInt(*req.OrderID, 10, 64)
			if parseErr != nil || orderID <= 0 {
				return nil, errors.New("invalid order_id")
			}
			if h.validateOrder == nil {
				response.Error(c, http.StatusServiceUnavailable, "order validation unavailable")
				return nil, errSupportResponseWritten
			}
			subject, _ := servermiddleware.GetAuthSubjectFromContext(c)
			if err := h.validateOrder(c.Request.Context(), subject.UserID, orderID); err != nil {
				if errors.Is(err, service.ErrSupportOrderForbidden) {
					response.Forbidden(c, "order belongs to another user")
				} else if errors.Is(err, service.ErrManagedRechargeOrderMissing) {
					response.NotFound(c, "order not found")
				} else {
					response.Error(c, http.StatusServiceUnavailable, "order validation unavailable")
				}
				return nil, errSupportResponseWritten
			}
		}
	case *supportReplyRequest:
		req.Content = strings.TrimSpace(req.Content)
		if req.Content == "" || utf8.RuneCountInString(req.Content) > 8000 || req.ClientMessageID == uuid.Nil || !validSupportAttachments(req.AttachmentIDs) {
			return nil, errors.New("invalid reply fields")
		}
	}
	return json.Marshal(payload)
}

var errSupportResponseWritten = errors.New("support response already written")

func validSupportType(value string) bool {
	switch value {
	case "pre-sale", "order", "after-sale", "technical", "other":
		return true
	}
	return false
}
func validSupportPriority(value string) bool {
	return value == "low" || value == "normal" || value == "high"
}
func validSupportAttachments(ids []uuid.UUID) bool {
	if len(ids) > 5 {
		return false
	}
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if id == uuid.Nil || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
