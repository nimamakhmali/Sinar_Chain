package dag

import "errors"

func (g *DAG) ValidateEvent(e *Event) error {
    if e.Hash != e.ComputeHash() {
        return errors.New("invalid event hash")
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

func (g *DAG) TopologicalOrder() ([]string, error) {
    visited := make(map[string]bool)
    tempMark := make(map[string]bool)
    order := []string{}

    var dfs func(hash string) error
    dfs = func(hash string) error {
        if tempMark[hash] {
            return errors.New("cycle detected")
        }
        if visited[hash] {
            return nil
        }
        tempMark[hash] = true
        for _, child := range g.children[hash] {
            if err := dfs(child); err != nil {
                return err
            }
        }
        tempMark[hash] = false
        visited[hash] = true
        order = append([]string{hash}, order...)
        return nil
    }

    for hash := range g.roots {
        if err := dfs(hash); err != nil {
            return nil, err
        }
    }
    return order, nil
}