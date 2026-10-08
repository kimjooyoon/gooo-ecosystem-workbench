package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func nativeEvidence(t *testing.T) map[string][]byte {
	return readNativeEvidence(t, "../../publication/intent-contrasts-20261008/native-evidence.tar.gz")
}

func readNativeEvidence(t *testing.T, path string) map[string][]byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	files := map[string][]byte{}
	for {
		h, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(h.Name, "/result.json") && !strings.HasSuffix(h.Name, "/replay.json") && !strings.HasSuffix(h.Name, "/original.gooo") && !strings.HasSuffix(h.Name, "/source.gooo") && !strings.HasSuffix(h.Name, "/generation.json") {
			continue
		}
		if h.Size > 2<<20 {
			t.Fatal("unexpected large observation")
		}
		raw, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		files[h.Name] = raw
	}
	return files
}

func TestPublishedNativeContrastValuesAndSavedReplay(t *testing.T) {
	files := nativeEvidence(t)
	compiler := "e0d046503939883330eb37f998a2e0e8fc5154e5"
	for _, family := range families {
		for requested := range uint16(8) {
			id := fmt.Sprintf("%s-mixed-0-request%d", family.name, requested)
			source := files[id+"/construction/original.gooo"]
			if _, err := nativeCases(source, family, requested); err != nil {
				t.Fatal(err)
			}
			for _, replay := range []bool{false, true} {
				name := "/result.json"
				if replay {
					name = "/replay.json"
				}
				if err := checkNative(files[id+name], source, family, requested, compiler, !replay); err != nil {
					t.Fatal(id, name, err)
				}
			}
		}
	}
	id := "filenames-mixed-0-request0"
	for _, mutate := range []func(*nativeExport){
		func(n *nativeExport) { n.Runtime.Calls = nil },
		func(n *nativeExport) { n.Runtime.Compiler = "different" },
		func(n *nativeExport) { n.Runtime.SourceSHA = "different" },
		func(n *nativeExport) { n.Runtime.Traces[0].Deliveries[0].Input = json.RawMessage(`"other"`) },
		func(n *nativeExport) { n.Runtime.Traces[0].Deliveries[0].Actual = json.RawMessage(`{}`) },
		func(n *nativeExport) { n.Composition.Steps[0].Generation.Report.Assembly.Calls = nil },
		func(n *nativeExport) { n.Runtime.Replay = false },
	} {
		var n nativeExport
		if err := json.Unmarshal(files[id+"/result.json"], &n); err != nil {
			t.Fatal(err)
		}
		mutate(&n)
		raw, _ := json.Marshal(n)
		if err := checkNative(raw, files[id+"/construction/original.gooo"], families[0], 0, compiler, true); err == nil {
			t.Fatal("incomplete or altered native observation accepted")
		}
	}
}
