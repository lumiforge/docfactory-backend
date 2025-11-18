package templates

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// NewInMemoryRepository creates thread-safe repository for prototyping.
func NewInMemoryRepository() Repository {
	return &inMemoryRepository{
		templates: make(map[string]Template),
		versions:  make(map[string][]TemplateVersion),
	}
}

// TemplateService orchestrates repository operations with validation and
// business logic.
type TemplateService struct {
	repo Repository
}

// NewTemplateService creates service instance.
func NewTemplateService(repo Repository) *TemplateService {
	return &TemplateService{repo: repo}
}

// CreateTemplate handles validation and creation.
func (s *TemplateService) CreateTemplate(ctx context.Context, tpl Template) (*Template, error) {
	tpl.TemplateID = newID()
	now := time.Now().UTC()
	tpl.CreatedAt = now
	tpl.UpdatedAt = now
	tpl.Version = 1
	if err := tpl.Validate(); err != nil {
		return nil, fmt.Errorf("validate template: %w", err)
	}
	created, err := s.repo.CreateTemplate(ctx, tpl)
	if err != nil {
		return nil, err
	}
	version := TemplateVersion{
		VersionID:     newID(),
		TemplateID:    tpl.TemplateID,
		VersionNumber: tpl.Version,
		JSONSchemaURL: tpl.JSONSchemaURL,
		ChangeSummary: "initial version",
		CreatedBy:     tpl.CreatedBy,
		CreatedAt:     now,
		IsCurrent:     true,
	}
	if _, err := s.repo.CreateVersion(ctx, tpl.TenantID, version); err != nil {
		return nil, err
	}
	return created, nil
}

// UpdateTemplate updates template metadata while incrementing version history.
func (s *TemplateService) UpdateTemplate(ctx context.Context, tenantID, templateID string, mutate func(*Template) error, updatedBy string, changeSummary string) (*Template, error) {
	tpl, err := s.repo.GetTemplate(ctx, tenantID, templateID)
	if err != nil {
		return nil, err
	}
	if tpl.DeletedAt != nil {
		return nil, fmt.Errorf("template is deleted: %w", ErrInvalidInput)
	}
	if err := mutate(tpl); err != nil {
		return nil, err
	}
	tpl.Version++
	tpl.UpdatedBy = updatedBy
	tpl.UpdatedAt = time.Now().UTC()
	if err := tpl.Validate(); err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateTemplate(ctx, *tpl)
	if err != nil {
		return nil, err
	}
	version := TemplateVersion{
		VersionID:     newID(),
		TemplateID:    tpl.TemplateID,
		VersionNumber: tpl.Version,
		JSONSchemaURL: tpl.JSONSchemaURL,
		ChangeSummary: changeSummary,
		CreatedBy:     updatedBy,
		CreatedAt:     tpl.UpdatedAt,
		IsCurrent:     true,
	}
	if _, err := s.repo.CreateVersion(ctx, tenantID, version); err != nil {
		return nil, err
	}
	return updated, nil
}

// DuplicateTemplate duplicates template with optional version copy.
func (s *TemplateService) DuplicateTemplate(ctx context.Context, tenantID, templateID string, opt DuplicateOptions) (*Template, error) {
	tpl, err := s.repo.DuplicateTemplate(ctx, tenantID, templateID, opt)
	if err != nil {
		return nil, err
	}
	if opt.CopyVersions {
		versions, err := s.repo.ListVersions(ctx, tenantID, templateID)
		if err != nil {
			return nil, err
		}
		for _, v := range versions {
			v.TemplateID = tpl.TemplateID
			v.VersionID = newID()
			if v.IsCurrent {
				v.VersionNumber = tpl.Version
			}
			if _, err := s.repo.CreateVersion(ctx, tenantID, v); err != nil {
				return nil, err
			}
		}
	}
	return tpl, nil
}

// RestoreTemplate performs soft delete restoration.
func (s *TemplateService) RestoreTemplate(ctx context.Context, tenantID, templateID string) (*Template, error) {
	return s.repo.RestoreTemplate(ctx, tenantID, templateID)
}

// DeleteTemplate performs soft delete.
func (s *TemplateService) DeleteTemplate(ctx context.Context, tenantID, templateID string) error {
	return s.repo.SoftDeleteTemplate(ctx, tenantID, templateID)
}

// ListTemplates proxies listing operation.
func (s *TemplateService) ListTemplates(ctx context.Context, opt ListOptions) ([]Template, int, error) {
	items, err := s.repo.ListTemplates(ctx, opt)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.CountTemplates(ctx, opt)
	if err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

// GetTemplate fetches template.
func (s *TemplateService) GetTemplate(ctx context.Context, tenantID, templateID string) (*Template, error) {
	return s.repo.GetTemplate(ctx, tenantID, templateID)
}

// Version helpers
func (s *TemplateService) ListVersions(ctx context.Context, tenantID, templateID string) ([]TemplateVersion, error) {
	return s.repo.ListVersions(ctx, tenantID, templateID)
}

func (s *TemplateService) RestoreVersion(ctx context.Context, tenantID, templateID string, versionNumber int) (*TemplateVersion, error) {
	return s.repo.RestoreVersion(ctx, tenantID, templateID, versionNumber)
}

func (s *TemplateService) CompareVersions(ctx context.Context, tenantID, templateID string, left, right int) (*VersionComparison, error) {
	return s.repo.CompareVersions(ctx, tenantID, templateID, left, right)
}

// inMemoryRepository is prototyping repository with maps.
type inMemoryRepository struct {
	templates map[string]Template
	versions  map[string][]TemplateVersion
	mu        sync.RWMutex
}

func (r *inMemoryRepository) withTenantTemplates(tenantID string) []Template {
	var res []Template
	for _, tpl := range r.templates {
		if tpl.TenantID == tenantID {
			res = append(res, tpl)
		}
	}
	return res
}

func (r *inMemoryRepository) ListTemplates(ctx context.Context, opt ListOptions) ([]Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []Template
	search := strings.ToLower(strings.TrimSpace(opt.Search))
	for _, tpl := range r.templates {
		if tpl.TenantID != opt.TenantID {
			continue
		}
		if !opt.IncludeDeleted && tpl.DeletedAt != nil {
			continue
		}
		if opt.DocumentType != "" && tpl.DocumentType != opt.DocumentType {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(tpl.Name), search) && !strings.Contains(strings.ToLower(tpl.Description), search) {
			continue
		}
		result = append(result, tpl)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	start := opt.Offset
	if start > len(result) {
		return []Template{}, nil
	}
	end := start + opt.Limit
	if opt.Limit <= 0 || end > len(result) {
		end = len(result)
	}
	return append([]Template(nil), result[start:end]...), nil
}

func (r *inMemoryRepository) CountTemplates(ctx context.Context, opt ListOptions) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	search := strings.ToLower(strings.TrimSpace(opt.Search))
	for _, tpl := range r.templates {
		if tpl.TenantID != opt.TenantID {
			continue
		}
		if !opt.IncludeDeleted && tpl.DeletedAt != nil {
			continue
		}
		if opt.DocumentType != "" && tpl.DocumentType != opt.DocumentType {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(tpl.Name), search) && !strings.Contains(strings.ToLower(tpl.Description), search) {
			continue
		}
		count++
	}
	return count, nil
}

func (r *inMemoryRepository) GetTemplate(ctx context.Context, tenantID, templateID string) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tpl, ok := r.templates[templateID]
	if !ok || tpl.TenantID != tenantID {
		return nil, ErrNotFound
	}
	clone := tpl
	return &clone, nil
}

func (r *inMemoryRepository) CreateTemplate(ctx context.Context, tpl Template) (*Template, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.templates[tpl.TemplateID]; exists {
		return nil, ErrConflict
	}
	r.templates[tpl.TemplateID] = tpl
	clone := tpl
	return &clone, nil
}

func (r *inMemoryRepository) UpdateTemplate(ctx context.Context, tpl Template) (*Template, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.templates[tpl.TemplateID]
	if !ok || existing.TenantID != tpl.TenantID {
		return nil, ErrNotFound
	}
	r.templates[tpl.TemplateID] = tpl
	clone := tpl
	return &clone, nil
}

func (r *inMemoryRepository) SoftDeleteTemplate(ctx context.Context, tenantID, templateID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	tpl, ok := r.templates[templateID]
	if !ok || tpl.TenantID != tenantID {
		return ErrNotFound
	}
	now := time.Now().UTC()
	tpl.DeletedAt = &now
	r.templates[templateID] = tpl
	return nil
}

func (r *inMemoryRepository) RestoreTemplate(ctx context.Context, tenantID, templateID string) (*Template, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tpl, ok := r.templates[templateID]
	if !ok || tpl.TenantID != tenantID {
		return nil, ErrNotFound
	}
	tpl.DeletedAt = nil
	r.templates[templateID] = tpl
	clone := tpl
	return &clone, nil
}

func (r *inMemoryRepository) DuplicateTemplate(ctx context.Context, tenantID, templateID string, opt DuplicateOptions) (*Template, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tpl, ok := r.templates[templateID]
	if !ok || tpl.TenantID != tenantID {
		return nil, ErrNotFound
	}
	now := time.Now().UTC()
	clone := tpl
	clone.TemplateID = newID()
	clone.CreatedAt = now
	clone.UpdatedAt = now
	clone.CreatedBy = opt.CreatedBy
	clone.UpdatedBy = opt.UpdatedBy
	clone.DeletedAt = nil
	clone.DocumentsCount = 0
	clone.LastUsedAt = nil
	clone.Version = 1
	if opt.NameOverride != "" {
		clone.Name = opt.NameOverride
	} else {
		clone.Name = fmt.Sprintf("%s Copy", tpl.Name)
	}
	if opt.DescriptionOverride != "" {
		clone.Description = opt.DescriptionOverride
	}
	if err := clone.Validate(); err != nil {
		return nil, err
	}
	if _, exists := r.templates[clone.TemplateID]; exists {
		return nil, ErrConflict
	}
	r.templates[clone.TemplateID] = clone
	version := TemplateVersion{
		VersionID:     newID(),
		TemplateID:    clone.TemplateID,
		VersionNumber: clone.Version,
		JSONSchemaURL: clone.JSONSchemaURL,
		ChangeSummary: "duplicated from " + tpl.TemplateID,
		CreatedBy:     opt.UpdatedBy,
		CreatedAt:     now,
		IsCurrent:     true,
	}
	r.versions[clone.TemplateID] = append(r.versions[clone.TemplateID], version)
	dup := clone
	return &dup, nil
}

func (r *inMemoryRepository) ListVersions(ctx context.Context, tenantID, templateID string) ([]TemplateVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tpl, ok := r.templates[templateID]
	if !ok || tpl.TenantID != tenantID {
		return nil, ErrNotFound
	}
	versions := r.versions[templateID]
	return append([]TemplateVersion(nil), versions...), nil
}

func (r *inMemoryRepository) CreateVersion(ctx context.Context, tenantID string, version TemplateVersion) (*TemplateVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tpl, ok := r.templates[version.TemplateID]
	if !ok || tpl.TenantID != tenantID {
		return nil, ErrNotFound
	}
	if err := version.Validate(); err != nil {
		return nil, err
	}
	for i := range r.versions[version.TemplateID] {
		r.versions[version.TemplateID][i].IsCurrent = false
	}
	r.versions[version.TemplateID] = append(r.versions[version.TemplateID], version)
	clone := version
	return &clone, nil
}

func (r *inMemoryRepository) RestoreVersion(ctx context.Context, tenantID, templateID string, versionNumber int) (*TemplateVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tpl, ok := r.templates[templateID]
	if !ok || tpl.TenantID != tenantID {
		return nil, ErrNotFound
	}
	versions := r.versions[templateID]
	var restored *TemplateVersion
	for i := range versions {
		if versions[i].VersionNumber == versionNumber {
			restored = &versions[i]
			break
		}
	}
	if restored == nil {
		return nil, ErrNotFound
	}
	tpl.JSONSchemaURL = restored.JSONSchemaURL
	tpl.Version++
	tpl.UpdatedAt = time.Now().UTC()
	tpl.UpdatedBy = restored.CreatedBy
	r.templates[templateID] = tpl
	clone := *restored
	clone.IsCurrent = true
	clone.VersionNumber = tpl.Version
	clone.CreatedAt = tpl.UpdatedAt
	r.versions[templateID] = append(r.versions[templateID], clone)
	return &clone, nil
}

func (r *inMemoryRepository) CompareVersions(ctx context.Context, tenantID, templateID string, left, right int) (*VersionComparison, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tpl, ok := r.templates[templateID]
	if !ok || tpl.TenantID != tenantID {
		return nil, ErrNotFound
	}
	var leftVersion, rightVersion *TemplateVersion
	for i := range r.versions[templateID] {
		version := r.versions[templateID][i]
		switch version.VersionNumber {
		case left:
			tmp := version
			leftVersion = &tmp
		case right:
			tmp := version
			rightVersion = &tmp
		}
	}
	if leftVersion == nil || rightVersion == nil {
		return nil, ErrNotFound
	}
	summary := fmt.Sprintf("left schema: %s, right schema: %s", leftVersion.JSONSchemaURL, rightVersion.JSONSchemaURL)
	return &VersionComparison{
		TemplateID: tpl.TemplateID,
		Left:       *leftVersion,
		Right:      *rightVersion,
		Summary:    summary,
	}, nil
}
