package dag

import(
	"fmt"
	"crypto"
)
type Frame struct {
	Events map[string]*Event
	Roots map[string]bool
}

type Consensus struct {
	dag           *DAG
	frames        []Frame
	lastAtropos   string
}

func (c *Consensus) DecideAtropos() {
	// Coming Soon
}