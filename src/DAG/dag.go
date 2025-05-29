package dag

import  "errors"
import 	"hash"


type DAG struct {
	events    map[string]*Event
	children  map[string][]string
	roots     map[string]bool
}

func NewDAG() *DAG {
	return &DAG{
		events   :  make(map[string]*Event),
		children :  make(map[string][]string),
		roots    :  make(map[string]bool),
	}
}

func (g *DAG) AddEvent(e *Event) error {
	if _, exists := g.events[e.Hash]; exists {
	return errors.New("event already exists")
	}

	g.events[e.Hash] = e

	for _, parent := range e.Parents {
		g.children[parent] = append(g.children[parent], e.Hash)
	}
	
	if len(e.Parents) == 0 {
		g.roots[e.Hash] = true
	}else {
		allKnown := true
		for _, p := range e.Parents {
			if _, exists := g.events[p]; !exists {
				allKnown = false
				break
			}
		}
		if !allKnown {
			g.roots[e.Hash] = true
		}
	}
	return nil
}

func (g *DAG) GetChildren(hash string) []string {
	return g.children[hash]
}

func (g *DAG) HasEvent(hash string) bool {
    _, exists := g.events[hash]
    return exists
}

func (g *DAG) ValidateEvent(e *Event) error {

    if e.Hash != e.ComputeHash() {
        return errors.New("invalid event hash")
    }

    for _, parent := range e.Parents {
        if !g.HasEvent(parent) {
            return errors.New("parent event not found")
        }
    }
    return nil
}

func (g *DAG) SelectParents() []string {
    var parents []string
    if len(g.roots) >= 2 {
        i := 0
        for hash := range g.roots {
            if i >= 2 {
                break
            }
            parents = append(parents, hash)
            i++
        }
    } else if len(g.roots) == 1 {
        for hash := range g.roots {
            parents = append(parents, hash)
            if len(g.children[hash]) > 0 {
                parents = append(parents, g.children[hash][0])
            }
        }
    }
    return parents
}