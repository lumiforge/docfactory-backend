package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lumiforge/docfactory-backend/internal/templates"
)

// TemplateHandler wires HTTP requests to template service.
type TemplateHandler struct {
	service *templates.TemplateService
}

// NewTemplateHandler creates HTTP handler.
func NewTemplateHandler(service *templates.TemplateService) *TemplateHandler {
	return &TemplateHandler{service: service}
}

// ListTemplates handles GET /templates.
func (h *TemplateHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	limit, offset := paginationFromRequest(r, 50)
	opt := templates.ListOptions{
		TenantID: tenantID,
		Search:   r.URL.Query().Get("search"),
		Limit:    limit,
		Offset:   offset,
	}
	if docType := r.URL.Query().Get("document_type"); docType != "" {
		opt.DocumentType = templates.DocumentType(docType)
	}
	includeDeleted := r.URL.Query().Get("include_deleted")
	opt.IncludeDeleted = includeDeleted == "true"

	templatesList, total, err := h.service.ListTemplates(r.Context(), opt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  templatesList,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetTemplate handles GET /templates/{id}.
func (h *TemplateHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	templateID := pathParam(r, "templateID")
	tpl, err := h.service.GetTemplate(r.Context(), tenantID, templateID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, templates.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, tpl)
}

// CreateTemplate handles POST /templates.
func (h *TemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	userID := userFromRequest(r)
	var payload TemplatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	tpl := payload.ToTemplate()
	tpl.TenantID = tenantID
	tpl.CreatedBy = userID
	tpl.UpdatedBy = userID
	created, err := h.service.CreateTemplate(r.Context(), tpl)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, templates.ErrInvalidInput) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// UpdateTemplate handles PUT /templates/{id}.
func (h *TemplateHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	userID := userFromRequest(r)
	templateID := pathParam(r, "templateID")
	var payload TemplatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, err := h.service.UpdateTemplate(r.Context(), tenantID, templateID, func(t *templates.Template) error {
		if payload.Name != "" {
			t.Name = payload.Name
		}
		t.Description = payload.Description
		if payload.DocumentType != "" {
			t.DocumentType = payload.DocumentType
		}
		if payload.PageSize != "" {
			t.PageSize = payload.PageSize
		}
		if payload.Orientation != "" {
			t.Orientation = payload.Orientation
		}
		if payload.JSONSchemaURL != "" {
			t.JSONSchemaURL = payload.JSONSchemaURL
		}
		if payload.ThumbnailURL != "" {
			t.ThumbnailURL = payload.ThumbnailURL
		}
		return nil
	}, userID, payload.ChangeSummary)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, templates.ErrInvalidInput) {
			status = http.StatusBadRequest
		} else if errors.Is(err, templates.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// DeleteTemplate handles DELETE /templates/{id}.
func (h *TemplateHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	templateID := pathParam(r, "templateID")
	if err := h.service.DeleteTemplate(r.Context(), tenantID, templateID); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, templates.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RestoreTemplate handles POST /templates/{id}/restore.
func (h *TemplateHandler) RestoreTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	templateID := pathParam(r, "templateID")
	tpl, err := h.service.RestoreTemplate(r.Context(), tenantID, templateID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, templates.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, tpl)
}

// DuplicateTemplate handles POST /templates/{id}/duplicate.
func (h *TemplateHandler) DuplicateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	templateID := pathParam(r, "templateID")
	var payload DuplicatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	payload.Defaults(userFromRequest(r))
	dup, err := h.service.DuplicateTemplate(r.Context(), tenantID, templateID, templates.DuplicateOptions{
		CreatedBy:           payload.CreatedBy,
		UpdatedBy:           payload.UpdatedBy,
		CopyVersions:        payload.CopyVersions,
		NameOverride:        payload.NameOverride,
		DescriptionOverride: payload.DescriptionOverride,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, templates.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusCreated, dup)
}

// ListVersions handles GET /templates/{id}/versions.
func (h *TemplateHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	templateID := pathParam(r, "templateID")
	versions, err := h.service.ListVersions(r.Context(), tenantID, templateID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, templates.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, versions)
}

// RestoreVersion handles POST /templates/{id}/versions/{version}/restore.
func (h *TemplateHandler) RestoreVersion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	templateID := pathParam(r, "templateID")
	versionParam := pathParam(r, "version")
	versionNumber, err := strconv.Atoi(versionParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("version must be integer"))
		return
	}
	restored, err := h.service.RestoreVersion(r.Context(), tenantID, templateID, versionNumber)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, templates.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, restored)
}

// CompareVersions handles GET /templates/{id}/versions/compare.
func (h *TemplateHandler) CompareVersions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	templateID := pathParam(r, "templateID")
	left, right := r.URL.Query().Get("left"), r.URL.Query().Get("right")
	leftVersion, err := strconv.Atoi(left)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("left is required integer"))
		return
	}
	rightVersion, err := strconv.Atoi(right)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("right is required integer"))
		return
	}
	comparison, err := h.service.CompareVersions(r.Context(), tenantID, templateID, leftVersion, rightVersion)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, templates.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, comparison)
}

// BulkDelete handles POST /templates/bulk/delete.
func (h *TemplateHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var payload BulkIDsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result := BulkResult{Succeeded: []string{}, Failed: map[string]string{}}
	for _, id := range payload.TemplateIDs {
		if err := h.service.DeleteTemplate(r.Context(), tenantID, id); err != nil {
			result.Failed[id] = err.Error()
		} else {
			result.Succeeded = append(result.Succeeded, id)
		}
	}
	writeJSON(w, http.StatusMultiStatus, result)
}

// BulkExport handles POST /templates/bulk/export.
func (h *TemplateHandler) BulkExport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	_ = tenantID
	var payload BulkIDsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	exportID := "export-" + strconv.FormatInt(time.Now().Unix(), 10)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"export_id":    exportID,
		"template_ids": payload.TemplateIDs,
		"status":       "scheduled",
	})
}

// BulkDuplicate handles POST /templates/bulk/duplicate.
func (h *TemplateHandler) BulkDuplicate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenantFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	userID := userFromRequest(r)
	var payload BulkDuplicatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result := BulkResult{Succeeded: []string{}, Failed: map[string]string{}}
	for _, id := range payload.TemplateIDs {
		dup, err := h.service.DuplicateTemplate(r.Context(), tenantID, id, templates.DuplicateOptions{
			CreatedBy:    userID,
			UpdatedBy:    userID,
			CopyVersions: payload.CopyVersions,
		})
		if err != nil {
			result.Failed[id] = err.Error()
			continue
		}
		result.Succeeded = append(result.Succeeded, dup.TemplateID)
	}
	writeJSON(w, http.StatusMultiStatus, result)
}

// Helper structures

type TemplatePayload struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	DocumentType  templates.DocumentType `json:"document_type"`
	PageSize      templates.PageSize     `json:"page_size"`
	Orientation   templates.Orientation  `json:"orientation"`
	JSONSchemaURL string                 `json:"json_schema_url"`
	ThumbnailURL  string                 `json:"thumbnail_url"`
	ChangeSummary string                 `json:"change_summary"`
}

func (p TemplatePayload) ToTemplate() templates.Template {
	return templates.Template{
		Name:          strings.TrimSpace(p.Name),
		Description:   strings.TrimSpace(p.Description),
		DocumentType:  p.DocumentType,
		PageSize:      p.PageSize,
		Orientation:   p.Orientation,
		JSONSchemaURL: strings.TrimSpace(p.JSONSchemaURL),
		ThumbnailURL:  strings.TrimSpace(p.ThumbnailURL),
	}
}

type DuplicatePayload struct {
	CopyVersions        bool   `json:"copy_versions"`
	NameOverride        string `json:"name_override"`
	DescriptionOverride string `json:"description_override"`
	CreatedBy           string `json:"created_by"`
	UpdatedBy           string `json:"updated_by"`
}

func (p *DuplicatePayload) Defaults(userID string) {
	if p.CreatedBy == "" {
		p.CreatedBy = userID
	}
	if p.UpdatedBy == "" {
		p.UpdatedBy = userID
	}
}

type BulkIDsPayload struct {
	TemplateIDs []string `json:"template_ids"`
}

type BulkDuplicatePayload struct {
	TemplateIDs  []string `json:"template_ids"`
	CopyVersions bool     `json:"copy_versions"`
}

type BulkResult struct {
	Succeeded []string          `json:"succeeded"`
	Failed    map[string]string `json:"failed"`
}

type contextKey string

func withPathParam(ctx context.Context, key, value string) context.Context {
	return context.WithValue(ctx, contextKey(key), value)
}

func pathParam(r *http.Request, key string) string {
	if value, ok := r.Context().Value(contextKey(key)).(string); ok {
		return value
	}
	return ""
}

func tenantFromRequest(r *http.Request) (string, error) {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		return "", errors.New("X-Tenant-ID header is required")
	}
	return tenantID, nil
}

func userFromRequest(r *http.Request) string {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		return "system"
	}
	return userID
}

func paginationFromRequest(r *http.Request, defaultLimit int) (int, int) {
	limit := defaultLimit
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return limit, offset
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
