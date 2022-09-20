package mission

import "fmt"

// Graph object represents the mission DAG in terms of links between stages
type Graph struct {
  down map[*Stage][]*Stage
  up   map[*Stage][]*Stage
}

// AddLink is used to create the graph object from each link seen in the graph.
func (g *Graph) AddLink(from *Stage, to *Stage) {
  if !stageListContains(g.down[from], to) { // prevent duplicates
    g.down[from] = append(g.down[from], to)
  }
  if !stageListContains(g.up[to], from) { // prevent duplicates
    g.up[to] = append(g.up[to], from)
  }
}

// CheckForCycle recursively crawls the graph and returns true if starting stage is seen again.
func (g *Graph) CheckForCycle(s *Stage, visited map[*Stage]bool, recursion map[*Stage]bool) bool {
  visited[s] = true
  recursion[s] = true // flag as recursive in this cycle so if we see it we know there's a cycle
  for _, downstreamStage := range g.down[s] {
    if !visited[downstreamStage] {
      if g.CheckForCycle(downstreamStage, visited, recursion) {
        return true
      }
      // else, finish visiting this node
    } else if recursion[downstreamStage] {
      return true // we saw the same node again --> graph is cyclic
    }
  }
  recursion[s] = false // we didn't see the same node again in this cycle, reset
  return false
}

// utility used by CheckForIncontiguity to visit every stage in the graph
func (g *Graph) visitRecursively(s *Stage, visited map[*Stage]bool) {
  if visited[s] {
    return // already visited - end loop
  }
  visited[s] = true
  for _, u := range g.up[s] {
    g.visitRecursively(u, visited)
  }
  for _, d := range g.down[s] {
    g.visitRecursively(d, visited)
  }
  return
}

// CheckForIncontiguity returns the first stage found that can't be reached from the starting stage.
// If the graph is contiguous then it will return nil.
func (g *Graph) CheckForIncontiguity(stages []*Stage) *Stage {
  startingStage := stages[0]
  visited := make(map[*Stage]bool)
  for _, s := range stages {
    visited[s] = false
  }
  g.visitRecursively(startingStage, visited) // ends when all stages have been visited
  for s, v := range visited {
    if !v {
      return s // a stage was not visited
    }
  }
  return nil
}

// recursive - if upstream stage is skipped then we need to look at it's dependencies - all must be finished or skipped
// if there are no upstream then always returns true
func (g *Graph) areUpstreamFinished(stage *Stage) bool {
  for _, upstreamStage := range g.up[stage] {
    // all upstream stages must be finished or skipped
    switch upstreamStage.State {
    case finished, excluded:
      continue
    case skipped:
      g.areUpstreamFinished(upstreamStage) // recurse
    default:
      return false
    }
  }
  return true
}

func (g *Graph) Print() {
  fmt.Println("looking downstream:")
  for stage, _ := range g.down {
    for _, downstreamStage := range g.down[stage] {
      fmt.Println("  " + stage.Name + " > " + downstreamStage.Name)
    }
  }
  fmt.Println("looking upstream:")
  for stage, _ := range g.up {
    for _, upstreamStage := range g.up[stage] {
      fmt.Println("  " + upstreamStage.Name + " > " + stage.Name)
    }
  }
}

// NewGraph builds graph object.
// note: This runs before we check that all stages are defined, so it just skips stages that can't be found.
func NewGraph(m *Mission) *Graph {

  // build graph object
  graph := &Graph{make(map[*Stage][]*Stage), make(map[*Stage][]*Stage)}

  // are all stages referred to in upstream/downstream defined?
  for _, s := range m.Stages {
    for _, u := range s.Upstream {
      // add to graph
      if upstreamStage, err := m.GetStage(u); err == nil {
        graph.AddLink(upstreamStage, s)
      }
    }
    for _, d := range s.Downstream {
      // add to graph
      if downstreamStage, err := m.GetStage(d); err == nil {
        graph.AddLink(s, downstreamStage)
      }
    }
  }

  return graph
}
