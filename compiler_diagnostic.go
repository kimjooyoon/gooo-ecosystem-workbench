package workbench

import (
	"encoding/json"
	"strings"
)

// Some compiler commands place their diagnostic in the JSON response on stdout.
func compilerFailureDetail(stderr string, stdout []byte) string {
	if detail := strings.TrimSpace(stderr); detail != "" {
		return detail
	}
	var response struct {
		Error   string `json:"error"`
		Failure string `json:"failure"`
	}
	if json.Unmarshal(stdout, &response) != nil {
		return ""
	}
	if detail := strings.TrimSpace(response.Error); detail != "" {
		return detail
	}
	return strings.TrimSpace(response.Failure)
}
