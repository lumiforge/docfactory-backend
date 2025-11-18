package templates

import (
	"errors"
	"strings"
	"time"
)

// DocumentType enumerates supported document types.
type DocumentType string

const (
	DocumentTypeWarranty    DocumentType = "warranty"
	DocumentTypeInstruction DocumentType = "instruction"
	DocumentTypeCertificate DocumentType = "certificate"
	DocumentTypeLabel       DocumentType = "label"
)

// PageSize enumerates supported document sizes.
type PageSize string

const (
	PageSizeA4     PageSize = "A4"
	PageSizeA5     PageSize = "A5"
	PageSizeLetter PageSize = "Letter"
)

// Orientation enumerates document orientation values.
type Orientation string

const (
	OrientationPortrait  Orientation = "portrait"
	OrientationLandscape Orientation = "landscape"
)

// Template represents the templates table structure.
type Template struct {
	TemplateID     string       `json:"template_id"`
	TenantID       string       `json:"tenant_id"`
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	DocumentType   DocumentType `json:"document_type"`
	PageSize       PageSize     `json:"page_size"`
	Orientation    Orientation  `json:"orientation"`
	JSONSchemaURL  string       `json:"json_schema_url"`
	ThumbnailURL   string       `json:"thumbnail_url"`
	Version        int          `json:"version"`
	CreatedBy      string       `json:"created_by"`
	UpdatedBy      string       `json:"updated_by"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	DeletedAt      *time.Time   `json:"deleted_at"`
	DocumentsCount int          `json:"documents_count"`
	LastUsedAt     *time.Time   `json:"last_used_at"`
}

// TemplateVersion represents the template_versions table structure.
type TemplateVersion struct {
	VersionID     string    `json:"version_id"`
	TemplateID    string    `json:"template_id"`
	VersionNumber int       `json:"version_number"`
	ChangeSummary string    `json:"change_summary"`
	JSONSchemaURL string    `json:"json_schema_url"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	IsCurrent     bool      `json:"is_current"`
}

var (
	// ErrNotFound is returned when template or version does not exist.
	ErrNotFound = errors.New("templates: resource not found")
	// ErrConflict is returned when duplicated operations conflict.
	ErrConflict = errors.New("templates: conflict detected")
	// ErrInvalidInput indicates validation error.
	ErrInvalidInput = errors.New("templates: invalid input")
)

// Validate ensures template structure is valid according to business rules.
func (t Template) Validate() error {
	if t.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if len(strings.TrimSpace(t.Name)) < 3 || len(t.Name) > 100 {
		return errors.New("name must be between 3 and 100 characters")
	}
	if len(t.Description) > 500 {
		return errors.New("description must be <= 500 characters")
	}
	switch t.DocumentType {
	case DocumentTypeWarranty, DocumentTypeInstruction, DocumentTypeCertificate, DocumentTypeLabel:
	default:
		return errors.New("document_type is invalid")
	}
	switch t.PageSize {
	case PageSizeA4, PageSizeA5, PageSizeLetter:
	default:
		return errors.New("page_size is invalid")
	}
	switch t.Orientation {
	case OrientationPortrait, OrientationLandscape:
	default:
		return errors.New("orientation is invalid")
	}
	if t.JSONSchemaURL == "" {
		return errors.New("json_schema_url is required")
	}
	if t.CreatedBy == "" {
		return errors.New("created_by is required")
	}
	if t.UpdatedBy == "" {
		return errors.New("updated_by is required")
	}
	return nil
}

// Validate ensures template version business rules.
func (tv TemplateVersion) Validate() error {
	if tv.TemplateID == "" {
		return errors.New("template_id is required")
	}
	if tv.VersionNumber <= 0 {
		return errors.New("version_number must be positive")
	}
	if tv.JSONSchemaURL == "" {
		return errors.New("json_schema_url is required")
	}
	if tv.CreatedBy == "" {
		return errors.New("created_by is required")
	}
	return nil
}
