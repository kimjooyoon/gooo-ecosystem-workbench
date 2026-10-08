package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
	workbench "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

type row struct {
	ID, Family, Language  string
	Order, Request        int
	Label                 uint16
	Full, Ablated, Legacy [jointdecision.ThreeFeatureDim]float32
}

func loadRows(path string) ([]row, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, "", err
	}
	defer gz.Close()
	raw, err := io.ReadAll(io.LimitReader(gz, 32<<20))
	if err != nil || len(raw) == 32<<20 {
		return nil, "", fmt.Errorf("bounded gzip input required: %v", err)
	}
	var input workbench.FeatureAuditInput
	if err = json.Unmarshal(raw, &input); err != nil {
		return nil, "", err
	}
	if input.FeatureVersion != jointdecision.RecordGraphSharedFeatureVersion || len(input.Cases) != 576 {
		return nil, "", fmt.Errorf("frozen graph corpus required")
	}
	rows := make([]row, len(input.Cases))
	seen := map[string]bool{}
	counts := map[string]int{}
	for i, c := range input.Cases {
		id := strings.Split(c.ID, "-")
		if len(id) != 4 || id[0] != c.Family || c.Graph == nil || len(c.AcceptedMasks) != 1 || seen[c.ID] || !strings.HasPrefix(id[3], "request") {
			return nil, "", fmt.Errorf("unique source graph and singleton label required")
		}
		seen[c.ID] = true
		order, e1 := strconv.Atoi(id[2])
		request, e2 := strconv.Atoi(strings.TrimPrefix(id[3], "request"))
		if e1 != nil || e2 != nil || order < 0 || order > 7 || request < 0 || request > 7 || c.AcceptedMasks[0] != uint16(order^request) ||
			id[1] != "ko" && id[1] != "en" && id[1] != "mixed" {
			return nil, "", fmt.Errorf("contrast design identity differs")
		}
		r := &rows[i]
		r.ID, r.Family, r.Language, r.Order, r.Request, r.Label = c.ID, c.Family, id[1], order, request, c.AcceptedMasks[0]
		text, err := jointdecision.EncodeRecordGraphThree(*c.Graph)
		if err != nil {
			return nil, "", err
		}
		if err = jointdecision.FeaturesIntoRecordGraphThree(text, &r.Full); err != nil {
			return nil, "", err
		}
		var legacy [3]jointdecision.RecordChoice
		graph := *c.Graph
		for bit, choice := range graph.Choices {
			legacy[bit] = choice.RecordChoice
			graph.Choices[bit].Intent = "intent withheld"
		}
		text, err = jointdecision.EncodeRecordThree(legacy)
		if err != nil {
			return nil, "", err
		}
		if err = jointdecision.FeaturesIntoRecordThree(text, &r.Legacy); err != nil {
			return nil, "", err
		}
		text, err = jointdecision.EncodeRecordGraphThree(graph)
		if err != nil {
			return nil, "", err
		}
		if err = jointdecision.FeaturesIntoRecordGraphThree(text, &r.Ablated); err != nil {
			return nil, "", err
		}
		counts[r.Family]++
	}
	if len(counts) != 3 || counts["filenames"] != 192 || counts["division"] != 192 || counts["retry"] != 192 {
		return nil, "", fmt.Errorf("three complete source families required")
	}
	return rows, digest(raw), nil
}

func split(rows []row, fold string) (train, test []int) {
	for i, r := range rows {
		if r.Family == fold {
			test = append(test, i)
		} else {
			train = append(train, i)
		}
	}
	return train, test
}

func digest(raw []byte) string { return fmt.Sprintf("%x", sha256.Sum256(raw)) }
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func save(path string, value any) {
	raw, err := json.MarshalIndent(value, "", "  ")
	must(err)
	must(os.WriteFile(path, append(raw, '\n'), 0644))
}
