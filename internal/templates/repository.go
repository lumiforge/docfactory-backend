package templates

import (
	"context"
)

// ListOptions configure search and pagination behaviour.
type ListOptions struct {
	TenantID       string
	Search         string
	DocumentType   DocumentType
	IncludeDeleted bool
	Limit          int
	Offset         int
}

// DuplicateOptions control template duplication behaviour.
type DuplicateOptions struct {
	CreatedBy           string
	UpdatedBy           string
	CopyVersions        bool
	NameOverride        string
	DescriptionOverride string
}

// Repository defines persistence layer for templates and versions.
type Repository interface {
	ListTemplates(ctx context.Context, opt ListOptions) ([]Template, error)
	CountTemplates(ctx context.Context, opt ListOptions) (int, error)
	GetTemplate(ctx context.Context, tenantID, templateID string) (*Template, error)
	CreateTemplate(ctx context.Context, tpl Template) (*Template, error)
	UpdateTemplate(ctx context.Context, tpl Template) (*Template, error)
	SoftDeleteTemplate(ctx context.Context, tenantID, templateID string) error
	RestoreTemplate(ctx context.Context, tenantID, templateID string) (*Template, error)
	DuplicateTemplate(ctx context.Context, tenantID, templateID string, opt DuplicateOptions) (*Template, error)

	ListVersions(ctx context.Context, tenantID, templateID string) ([]TemplateVersion, error)
	CreateVersion(ctx context.Context, tenantID string, version TemplateVersion) (*TemplateVersion, error)
	RestoreVersion(ctx context.Context, tenantID, templateID string, versionNumber int) (*TemplateVersion, error)
	CompareVersions(ctx context.Context, tenantID, templateID string, left, right int) (*VersionComparison, error)
}

// VersionComparison describes the difference between two versions. For now it
// only returns metadata but can be extended to include structural diff.
type VersionComparison struct {
	TemplateID string          `json:"template_id"`
	Left       TemplateVersion `json:"left"`
	Right      TemplateVersion `json:"right"`
	Summary    string          `json:"summary"`
}
