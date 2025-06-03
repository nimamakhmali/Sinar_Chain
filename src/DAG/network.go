package dag

import (
	dag "_/D_/Sinar/Sinar_Chain/src/DAG"
	"crypto"
	"encoding/gob"
	"host"
)

type Node struct {
	host         host.Host
	dag    		*DAG
	consensus   *Consensus
}

func NewNode() *Node {
	priv, _ := crypto.GenerateKey()
	h, _    := libp2p.New(libp2p.Identity(priv))
	return &Node {
		host: h,
		dag: NewDAG(),
	}
}

func (n *Node) handleEventStream(s network.Stream) {
	decoder := gob.NewDecoder(s)
	var event dag.Event
	decoder.Decode(&even)
	if n.dag.ValidateEvent(&event) == nil {
		n.dag.AddEvent(&event)
		n.consensus.ProcessEvent(&event)
	}
}
