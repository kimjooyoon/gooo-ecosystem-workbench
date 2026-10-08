package workbench

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"unicode/utf8"
)

// SpliceRequest names the complete original context, including the occurrence
// to replace. Every string follows Gooo's 1024-byte Text boundary.
type SpliceRequest struct {
	Source      string `json:"source"`
	Before      string `json:"before"`
	Expected    string `json:"expected"`
	After       string `json:"after"`
	Replacement string `json:"replacement"`
}

type SpliceEdit struct {
	Changed bool   `json:"changed"`
	Source  string `json:"source"`
	Bytes   int64  `json:"bytes"`
}

type SpliceObservation struct {
	Schema      string     `json:"schema"`
	Edit        SpliceEdit `json:"edit"`
	RequestSHA  string     `json:"request_sha256"`
	Observation Summary    `json:"observation"`
	Scope       string     `json:"scope"`
}

func ReadSpliceRequest(raw []byte) (SpliceRequest, error) {
	var request SpliceRequest
	if len(raw) == 0 || len(raw) > 32<<10 || !utf8.Valid(raw) {
		return request, fmt.Errorf("splice request requires at most 32 KiB of UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return request, fmt.Errorf("one splice request required")
	}
	// Empty values have meaning, so require explicit keys rather than treating
	// an accidentally omitted expected fragment as an insertion request.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return request, err
	}
	for _, name := range []string{"source", "before", "expected", "after", "replacement"} {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return request, fmt.Errorf("splice requires explicit string %s", name)
		}
	}
	return request, request.validate()
}

func (r SpliceRequest) validate() error {
	for _, text := range []string{r.Source, r.Before, r.Expected, r.After, r.Replacement} {
		if len(text) > 1024 || !utf8.ValidString(text) {
			return fmt.Errorf("splice Text values require at most 1024 UTF-8 bytes")
		}
	}
	return nil
}

// SpliceSource executes the Gooo-owned edit decision and sequential field
// updates. Go transports the request and publishes the observed output.
func SpliceSource(ctx context.Context, o Options, request SpliceRequest) (SpliceObservation, error) {
	var report SpliceObservation
	if err := request.validate(); err != nil {
		return report, err
	}
	root, err := newOutput(o.Out)
	if err != nil {
		return report, err
	}
	model, err := prepareModel(o.Model, root)
	if err != nil {
		return report, err
	}
	input, err := json.Marshal(request)
	if err != nil {
		return report, err
	}
	if err = write(filepath.Join(root, "request.json"), input); err != nil {
		return report, err
	}
	raw, built, err := executeSpliceRequest(ctx, o, root, model, request, false)
	if err != nil {
		return report, err
	}
	summary, err := summarize(raw, "splice", map[bool]string{true: "model", false: "deterministic"}[model != ""])
	if err != nil {
		return report, err
	}
	if summary.SelectionPassed != 24 || summary.SelectionTotal != 24 || len(built.Runtime.Traces) != 1 || len(built.Runtime.Traces[0].Deliveries) != 1 {
		return report, fmt.Errorf("splice recipe did not complete its declared selection cases")
	}
	if err := spliceOutput(built, &report.Edit); err != nil {
		return report, err
	}
	if report.Edit.Bytes != int64(len(report.Edit.Source)) || len(report.Edit.Source) > 1024 || !utf8.ValidString(report.Edit.Source) {
		return report, fmt.Errorf("Gooo splice output has inconsistent text accounting")
	}
	_, saved, err := executeSpliceRequest(ctx, o, root, "", request, true)
	if err != nil {
		return report, err
	}
	var edit SpliceEdit
	if err = spliceOutput(saved, &edit); err != nil {
		return report, err
	}
	if saved.Generated || saved.Runtime.Calls != 0 || saved.Composition.GeneratedSHA != built.Composition.GeneratedSHA || edit != report.Edit {
		return report, fmt.Errorf("saved splice replay differs")
	}
	summary.ReplayVerified = true
	report.Schema, report.Observation = "gooo/source-splice/v1", summary
	report.RequestSHA = fmt.Sprintf("sha256:%x", sha256.Sum256(input))
	report.Scope = "Gooo exact-context source fragment edit; each Text and resulting source bounded to 1024 UTF-8 bytes. Declared selection cases scored separately; this request has observed output and saved replay, without a supplied external expected value."
	if err = write(filepath.Join(root, "edited.txt"), []byte(report.Edit.Source)); err != nil {
		return report, err
	}
	return report, save(filepath.Join(root, "splice.json"), report)
}

func spliceOutput(r result, edit *SpliceEdit) error {
	if len(r.Composition.Steps) != 1 || len(r.Runtime.Traces) != 1 || len(r.Runtime.Traces[0].Deliveries) != 1 {
		return fmt.Errorf("splice requires exactly one constructed activity and one observed output")
	}
	delivery := r.Runtime.Traces[0].Deliveries[0]
	if delivery.ID == "" || delivery.ID != r.Composition.Steps[0].Generation.Report.ActivityID {
		return fmt.Errorf("splice output identity differs from its constructed activity")
	}
	return json.Unmarshal(delivery.Actual, edit)
}
