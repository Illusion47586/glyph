package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Graph struct {
	GeneratedAt string         `json:"generated_at"`
	Root        string         `json:"root"`
	Nodes       []GraphNode    `json:"nodes"`
	Edges       []GraphEdge    `json:"edges"`
	Summary     map[string]int `json:"summary"`
}

type GraphNode struct {
	ID      string            `json:"id"`
	Type    string            `json:"type"`
	Label   string            `json:"label"`
	Details map[string]string `json:"details,omitempty"`
}

type GraphEdge struct {
	From    string            `json:"from"`
	To      string            `json:"to"`
	Type    string            `json:"type"`
	Details map[string]string `json:"details,omitempty"`
}

func (s *Store) MeshGraph() (*Graph, error) {
	g := &Graph{
		GeneratedAt: nowUTC(),
		Root:        s.Root,
		Nodes:       []GraphNode{},
		Edges:       []GraphEdge{},
		Summary:     map[string]int{},
	}
	g.addNode("store:"+s.Root, "store", filepath.Base(s.Root), map[string]string{"root": s.Root, "store": s.Dir})
	if err := s.addRealmGraph(g); err != nil {
		return nil, err
	}
	if err := s.addSourceGraph(g); err != nil {
		return nil, err
	}
	if err := s.addWorkGraph(g); err != nil {
		return nil, err
	}
	if err := s.addPublicationGraph(g); err != nil {
		return nil, err
	}
	if err := s.addConcurrencyGraph(g); err != nil {
		return nil, err
	}
	if err := s.addHookGraph(g); err != nil {
		return nil, err
	}
	if err := s.addRemoteMountGraph(g); err != nil {
		return nil, err
	}
	g.sort()
	return g, nil
}

func (g *Graph) addNode(id, typ, label string, details map[string]string) {
	g.Nodes = append(g.Nodes, GraphNode{ID: id, Type: typ, Label: label, Details: details})
	g.Summary[typ]++
}

func (g *Graph) addEdge(from, to, typ string, details map[string]string) {
	g.Edges = append(g.Edges, GraphEdge{From: from, To: to, Type: typ, Details: details})
}

func (g *Graph) sort() {
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool {
		if g.Edges[i].From == g.Edges[j].From {
			if g.Edges[i].To == g.Edges[j].To {
				return g.Edges[i].Type < g.Edges[j].Type
			}
			return g.Edges[i].To < g.Edges[j].To
		}
		return g.Edges[i].From < g.Edges[j].From
	})
}

func (s *Store) WriteVisualizer(out string) (*Graph, error) {
	g, err := s.MeshGraph()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(out, "graph.json"), append(data, '\n'), 0o644); err != nil {
		return nil, err
	}
	html := strings.Replace(visualizerHTML, "__GLYPH_GRAPH_JSON__", string(data), 1)
	if err := os.WriteFile(filepath.Join(out, "index.html"), []byte(html), 0o644); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Store) addRealmGraph(g *Graph) error {
	rows, err := s.DB.Query(`SELECT name, description FROM realms ORDER BY name`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name, desc string
		if err := rows.Scan(&name, &desc); err != nil {
			return err
		}
		id := "realm:" + name
		g.addNode(id, "realm", name, map[string]string{"description": desc})
		g.addEdge("store:"+s.Root, id, "contains", nil)
	}
	return rows.Err()
}

func (s *Store) addSourceGraph(g *Graph) error {
	rows, err := s.DB.Query(`SELECT s.path, s.content_id, s.labels, s.updated_at, c.hash, c.size FROM sources s JOIN content c ON c.id = s.content_id ORDER BY s.path`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var path, contentID, labels, updated, hash string
		var size int64
		if err := rows.Scan(&path, &contentID, &labels, &updated, &hash, &size); err != nil {
			return err
		}
		sourceID := "source:" + path
		g.addNode(sourceID, "source", path, map[string]string{"labels": labels, "updated_at": updated})
		g.addNode(contentID, "content", shortHash(hash), map[string]string{"hash": hash, "size": int64String(size)})
		g.addEdge(sourceID, contentID, "points_to", nil)
		for _, label := range strings.Split(labels, ",") {
			label = strings.TrimSpace(label)
			if label != "" {
				g.addEdge("realm:"+label, sourceID, "contains", nil)
			}
		}
	}
	return rows.Err()
}
