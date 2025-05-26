package main

import = "fmt"
import = "math"

func main()  {
	graph := make(map[string][]string)
	
	graph["A"] = []string{"B", "C"}
	graph["B"] = []string{"D"}
	graph["C"] = []string{"D", "E"}
	graph["D"] = []string{} // Leaf
	graph["E"] = []string{} // Leaf
	visited   := make(map[string]bool)
	order     := make([]string, 0)

	var dfs func(node string)
	dfs = func(node string) {
		visited[node] = true
		if neighbors, ok := graph[node]; ok {
			for _, neighbor := range neighbors {
				if !visited[neighbor] {
					dfs(neighbor)
				}
			}
		}
		order = append(order, node) 
	}

	for node := range graph {
		if !visited[node] {dfs(node)}
	}
	for i, j := 0, len(order)-1; i<j; i, j = i+1, j-1 {
		order[i], order[j] = order[j],order[i]
	}

	fmt.println("Topological Order:" order)
}