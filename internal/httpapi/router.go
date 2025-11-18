package httpapi

import (
	"net/http"
	"strings"
)

// Router builds HTTP handler using net/http without external deps.
func Router(handler *TemplateHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		if path == "" {
			http.NotFound(w, r)
			return
		}
		segments := strings.Split(path, "/")
		if segments[0] != "templates" {
			http.NotFound(w, r)
			return
		}
		switch {
		case len(segments) == 1:
			handleTemplatesCollection(handler, w, r)
		case len(segments) >= 2:
			if segments[1] == "bulk" {
				handleTemplatesBulk(handler, w, r, segments[2:])
				return
			}
			ctx := withPathParam(r.Context(), "templateID", segments[1])
			if len(segments) == 2 {
				handlerTemplate(handler, w, r.WithContext(ctx))
				return
			}
			switch segments[2] {
			case "restore":
				handler.RestoreTemplate(w, r.WithContext(ctx))
			case "duplicate":
				handler.DuplicateTemplate(w, r.WithContext(ctx))
			case "versions":
				handleVersions(handler, w, r.WithContext(ctx), segments[3:])
			default:
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	})
}

func handleTemplatesCollection(handler *TemplateHandler, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handler.ListTemplates(w, r)
	case http.MethodPost:
		handler.CreateTemplate(w, r)
	default:
		methodNotAllowed(w)
	}
}

func handlerTemplate(handler *TemplateHandler, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handler.GetTemplate(w, r)
	case http.MethodPut:
		handler.UpdateTemplate(w, r)
	case http.MethodDelete:
		handler.DeleteTemplate(w, r)
	default:
		methodNotAllowed(w)
	}
}

func handleTemplatesBulk(handler *TemplateHandler, w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 0 {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	switch segments[0] {
	case "delete":
		handler.BulkDelete(w, r)
	case "export":
		handler.BulkExport(w, r)
	case "duplicate":
		handler.BulkDuplicate(w, r)
	default:
		http.NotFound(w, r)
	}
}

func handleVersions(handler *TemplateHandler, w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 0 {
		if r.Method == http.MethodGet {
			handler.ListVersions(w, r)
			return
		}
		methodNotAllowed(w)
		return
	}
	switch segments[0] {
	case "compare":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		handler.CompareVersions(w, r)
	default:
		if len(segments) == 2 && segments[1] == "restore" && r.Method == http.MethodPost {
			ctx := withPathParam(r.Context(), "version", segments[0])
			handler.RestoreVersion(w, r.WithContext(ctx))
			return
		}
		http.NotFound(w, r)
	}
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
