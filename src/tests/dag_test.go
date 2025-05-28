package tests

import "testing"
import "time"
import "Sinar_Chain/src/DAG"

func TestEventCreation(t *testing.T) {
	e := dag.Event {
		Creator   : "Node_1",
		Index     : 1,
		Parents   : []string{"genesis"},
		Timestamp : time.Now(),
		TxData    : [][]byte{[]byte("tx1")},
	}
	e.Hash = e.ComputeHash()
	t.Logf("Created Event: %+v", e)
}

func TestAddEventToDAG(t *testing.T) {
	g := dag.NewDAG()

	// Genesis event
	e1 := &dag.Event{
		Creator:   "Node_1",
		Index:     0,
		Parents:   []string{},
		Timestamp: time.Now(),
		TxData:    [][]byte{},
	}
	e1.Hash = e1.ComputeHash()
	_ = g.AddEvent(e1)

	// Event دوم با parent = e1
	e2 := &dag.Event{
		Creator:   "Node_1",
		Index:     1,
		Parents:   []string{e1.Hash},
		Timestamp: time.Now(),
		TxData:    [][]byte{[]byte("tx1")},
	}
	e2.Hash = e2.ComputeHash()
	err := g.AddEvent(e2)

	if err != nil {
		t.Errorf("AddEvent failed: %v", err)
	}

	children := g.GetChildren(e1.Hash)
	if len(children) != 1 || children[0] != e2.Hash {
		t.Errorf("Child linkage failed")
	}

	t.Logf("DAG: event %s added with parent %s", e2.Hash, e1.Hash)
}
