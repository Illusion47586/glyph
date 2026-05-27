package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMeshGraphIncludesCoreNodesAndEdges(t *testing.T) {
	_, st := newTestStore(t)
	defer st.Close()
	if _, err := st.ImportWorkspace(); err != nil {
		t.Fatalf("import: %v", err)
	}
	if _, err := st.StartWork("viz-work", "public"); err != nil {
		t.Fatalf("start work: %v", err)
	}
	if _, err := st.ClaimWork("viz-work", "agent:codex:viz", "exclusive", 0); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if _, err := st.PublishWithMode("viz-work", "public", "squash"); err != nil {
		t.Fatalf("publish: %v", err)
	}

	graph, err := st.MeshGraph()
	if err != nil {
		t.Fatalf("mesh graph: %v", err)
	}
	for _, typ := range []string{"store", "realm", "work", "snapshot", "publication", "source", "content", "claim"} {
		if graph.Summary[typ] == 0 {
			t.Fatalf("summary[%s] = 0; summary=%#v", typ, graph.Summary)
		}
	}
	if !hasEdge(graph, "work:viz-work", "realm:public", "based_on") {
		t.Fatalf("missing work based_on edge")
	}
	if !hasEdgeType(graph, "published_to") {
		t.Fatalf("missing published_to edge")
	}
	for _, typ := range []string{"work_started", "snapshot_created", "publication_published", "work_claimed"} {
		if !hasEventType(graph, typ) {
			t.Fatalf("missing event type %s; events=%#v", typ, graph.Events)
		}
	}
	if !eventsSorted(graph.Events) {
		t.Fatalf("events are not sorted by timestamp: %#v", graph.Events)
	}
	if !eventReferencesNode(graph, "publication_published", "work:viz-work") {
		t.Fatalf("publication event does not reference work node")
	}
}

func TestVisualizerExportIncludesTimeline(t *testing.T) {
	root, st := newTestStore(t)
	defer st.Close()
	if _, err := st.ImportWorkspace(); err != nil {
		t.Fatalf("import: %v", err)
	}
	if _, err := st.StartWork("timeline-work", "public"); err != nil {
		t.Fatalf("start work: %v", err)
	}
	out := filepath.Join(root, "viz")
	graph, err := st.WriteVisualizer(out)
	if err != nil {
		t.Fatalf("write visualizer: %v", err)
	}
	if len(graph.Events) == 0 {
		t.Fatalf("graph events empty")
	}
	html, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("read visualizer html: %v", err)
	}
	for _, want := range []string{`"events"`, "Timeline", "drawTimeline", "selectedEvent"} {
		if !strings.Contains(string(html), want) {
			t.Fatalf("visualizer html missing %q", want)
		}
	}
}

func hasEdge(graph *Graph, from, to, typ string) bool {
	for _, edge := range graph.Edges {
		if edge.From == from && edge.To == to && edge.Type == typ {
			return true
		}
	}
	return false
}

func hasEdgeType(graph *Graph, typ string) bool {
	for _, edge := range graph.Edges {
		if edge.Type == typ {
			return true
		}
	}
	return false
}

func hasEventType(graph *Graph, typ string) bool {
	for _, event := range graph.Events {
		if event.Type == typ {
			return true
		}
	}
	return false
}

func eventsSorted(events []GraphEvent) bool {
	for i := 1; i < len(events); i++ {
		if events[i-1].Timestamp > events[i].Timestamp {
			return false
		}
	}
	return true
}

func eventReferencesNode(graph *Graph, typ, nodeID string) bool {
	for _, event := range graph.Events {
		if event.Type != typ {
			continue
		}
		for _, id := range event.NodeIDs {
			if id == nodeID {
				return true
			}
		}
	}
	return false
}
