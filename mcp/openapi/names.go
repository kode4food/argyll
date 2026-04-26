package openapi

import (
	"regexp"
	"strings"

	openapi "github.com/getkin/kin-openapi/openapi3"

	"github.com/kode4food/argyll/engine/pkg/util"
)

var (
	actionWords = util.SetOf(
		"by", "create", "delete", "find", "list", "lookup", "new", "search",
		"update", "upsert",
	)

	versionSegment = regexp.MustCompile(`^v[0-9]+$`)
	camelBoundary  = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	nonWord        = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

func inferOperationID(op *openapi.Operation, method, path string) string {
	if op.OperationID != "" {
		return slug(op.OperationID)
	}
	return slug(strings.ToLower(method) + "-" + strings.Trim(path, "/"))
}

func inferEntity(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if part == "" || isParamSegment(part) ||
			versionSegment.MatchString(part) {
			continue
		}
		part = slugWord(part)
		if strings.HasPrefix(part, "by_") {
			continue
		}
		if actionWords.Contains(part) {
			continue
		}
		return singularize(part)
	}
	return ""
}

func inferParamEntity(path, name string) string {
	if name != "id" {
		return ""
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if !isParamSegment(part) || strings.Trim(part, "{}") != name {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			prev := slugWord(parts[j])
			if prev == "" || versionSegment.MatchString(prev) {
				continue
			}
			if actionWords.Contains(prev) {
				continue
			}
			return singularize(prev)
		}
	}
	return ""
}

func canonicalName(name, entity string) string {
	n := slugWord(name)
	switch {
	case n == "id" && entity != "":
		return entity + "_id"
	case strings.HasSuffix(n, "_id"):
		return n
	case strings.HasSuffix(n, "id") && len(n) > 2:
		n = strings.TrimSuffix(n, "id") + "_id"
		n = strings.Trim(n, "_")
	case entity != "" && n == "email":
		return entity + "_email"
	}
	return n
}

func singularize(s string) string {
	switch {
	case strings.HasSuffix(s, "ies") && len(s) > 3:
		return strings.TrimSuffix(s, "ies") + "y"
	case strings.HasSuffix(s, "ses") && len(s) > 3:
		return strings.TrimSuffix(s, "es")
	case strings.HasSuffix(s, "s") && len(s) > 1:
		return strings.TrimSuffix(s, "s")
	default:
		return s
	}
}

func pluralName(s string) string {
	if strings.HasSuffix(s, "s") {
		return s
	}
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		return strings.TrimSuffix(s, "y") + "ies"
	}
	return s + "s"
}

func requiredRole(required bool) string {
	if required {
		return "required"
	}
	return "optional"
}

func confidence(name, entity string) string {
	if name == "id" && entity != "" {
		return "high"
	}
	if strings.Contains(strings.ToLower(name), "id") {
		return "high"
	}
	if entity != "" {
		return "medium"
	}
	return "low"
}

func shouldExposeOutputProp(name, entity string) bool {
	canon := canonicalName(name, entity)
	if strings.HasSuffix(canon, "_id") {
		return true
	}
	return canon == "status"
}

func isWrapperProp(name string) bool {
	switch slugWord(name) {
	case "data", "result", "payload", "item":
		return true
	default:
		return false
	}
}

func slug(s string) string {
	s = camelBoundary.ReplaceAllString(s, "${1}-${2}")
	s = nonWord.ReplaceAllString(strings.ToLower(s), "-")
	s = strings.Trim(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}

func slugWord(s string) string {
	return strings.ReplaceAll(slug(s), "-", "_")
}

func isParamSegment(s string) bool {
	return strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
}
