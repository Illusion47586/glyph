package store

import "testing"

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
