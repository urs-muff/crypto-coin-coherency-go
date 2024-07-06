package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CID represents a Content Identifier in IPFS
type CID string

// PeerID represents a unique identifier for a peer in the network
type PeerID string

// ConceptGUID represents a globally unique identifier
type GUID string
type ConceptGUID GUID

type ConceptGUID2CIDMap map[ConceptGUID]CID

// Peer_i represents a peer in the network
type Peer_i interface {
	GetID() PeerID
	GetOwnerGUID() InstanceGUID

	AddConceptCID(cid CID)
	RemoveConceptCID(cid CID)
	GetConceptCIDs() []CID

	AddInstanceCID(cid CID)
	RemoveInstanceCID(cid CID)
	GetInstanceCIDs() []CID

	GetTimestamp() time.Time
}

// Network_i defines the interface for interacting with the network
type Network_i interface {
	// Add adds content to the network and returns its CID
	Add(ctx context.Context, content io.Reader) (CID, error)

	// Get retrieves content from the network by its CID
	Get(ctx context.Context, cid CID) (io.ReadCloser, error)

	// Remove removes content from the network by its CID
	Remove(ctx context.Context, cid CID) error

	// List returns a list of all CIDs stored by this node
	List(ctx context.Context) ([]CID, error)

	// Load loads data from a given path in the network
	Load(ctx context.Context, path string, target interface{}) error

	// Save saves data to a given path in the network
	Save(ctx context.Context, path string, data interface{}) error

	// Publish publishes a message to a topic
	Publish(ctx context.Context, topic string, data []byte) error

	// Subscribe subscribes to a topic and returns a channel for receiving messages
	Subscribe(ctx context.Context, topic string) (<-chan []byte, error)

	// Connect connects to a peer
	Connect(ctx context.Context, peerID PeerID) error

	// ListPeers returns a list of connected peers
	ListPeers(ctx context.Context) ([]Peer_i, error)
}

// Node_i represents a node in the network
type Node_i interface {
	Network_i

	// Bootstrap connects to bootstrap nodes
	Bootstrap(ctx context.Context) error

	// ID returns the ID of this node
	ID(ctx context.Context) (PeerID, error)
}

// Now let's define some concrete implementations of these interfaces

// Concept implements the Concept_i interface
type Concept struct {
	CID           CID `json:"-"`
	ID            ConceptGUID
	Name          string
	Description   string
	Type          string
	Relationships []ConceptGUID
	Timestamp     time.Time
}

type ConceptMap map[ConceptGUID]*Concept

func (c Concept) GetCID() CID                     { return c.CID }
func (c Concept) GetGUID() ConceptGUID            { return c.ID }
func (c Concept) GetName() string                 { return c.Name }
func (c Concept) GetDescription() string          { return c.Description }
func (c Concept) GetType() string                 { return c.Type }
func (c Concept) GetRelationships() []ConceptGUID { return c.Relationships }
func (c Concept) GetTimestamp() time.Time         { return c.Timestamp }
func (c Concept) String() string                  { return fmt.Sprintf("%s: %s (%s)", c.ID, c.Name, c.Type) }

// Relationship represents a connection between two entities
type Relationship struct {
	ID              ConceptGUID
	SourceID        ConceptGUID
	TargetID        ConceptGUID
	Type            ConceptGUID
	EnergyFlow      float64
	FrequencySpec   []float64
	Amplitude       float64
	Volume          float64
	Depth           int
	Interactions    int
	LastInteraction time.Time
	Timestamp       time.Time
}

func (r Relationship) String() string {
	return fmt.Sprintf("%s (%s): [%s] => [%s]",
		r.ID.AsConcept(),
		r.Type.AsConcept(),
		r.SourceID.AsConcept(),
		r.TargetID.AsConcept())
}

// RelationshipMap stores all relationships
type RelationshipMap map[ConceptGUID]*Relationship

// Function to create a new relationship
func CreateRelationship(sourceID, targetID ConceptGUID, relationType ConceptGUID) *Relationship {
	return &Relationship{
		ID:            ConceptGUID(uuid.New().String()),
		SourceID:      sourceID,
		TargetID:      targetID,
		Type:          relationType,
		EnergyFlow:    1.0, // Initial values, can be adjusted
		FrequencySpec: []float64{1.0},
		Amplitude:     1.0,
		Volume:        1.0,
		Depth:         1,
		Interactions:  0,
		Timestamp:     time.Now(),
	}
}

// Function to update a relationship
func (r *Relationship) Deepen() {
	r.EnergyFlow *= 1.1
	r.Amplitude *= 1.05
	r.Volume *= 1.05
	r.FrequencySpec = append(r.FrequencySpec, float64(len(r.FrequencySpec)+1))
	r.Timestamp = time.Now()
}

// ConcretePeer implements the Peer_i interface
type Peer struct {
	ID           PeerID
	OwnerGUID    InstanceGUID
	ConceptCIDs  map[CID]bool
	InstanceCIDs map[CID]bool
	Timestamp    time.Time
}

func (p Peer) GetID() PeerID              { return p.ID }
func (p Peer) GetOwnerGUID() InstanceGUID { return p.OwnerGUID }

func (p *Peer) AddConceptCID(cid CID)    { p.ConceptCIDs[cid] = true }
func (p *Peer) RemoveConceptCID(cid CID) { delete(p.ConceptCIDs, cid) }
func (p Peer) GetConceptCIDs() []CID {
	ret := make([]CID, 0)
	for cid := range p.ConceptCIDs {
		ret = append(ret, cid)
	}
	return ret
}

func (p *Peer) AddInstanceCID(cid CID)    { p.InstanceCIDs[cid] = true }
func (p *Peer) RemoveInstanceCID(cid CID) { delete(p.InstanceCIDs, cid) }
func (p Peer) GetInstanceCIDs() []CID {
	ret := make([]CID, 0)
	for cid := range p.InstanceCIDs {
		ret = append(ret, cid)
	}
	return ret
}

func (p Peer) GetTimestamp() time.Time { return p.Timestamp }

func (p *Peer) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID           PeerID
		OwnerGUID    InstanceGUID
		ConceptCIDs  []CID
		InstanceCIDs []CID
		Timestamp    time.Time
	}{
		ID:           p.ID,
		OwnerGUID:    p.OwnerGUID,
		ConceptCIDs:  p.GetConceptCIDs(),
		InstanceCIDs: p.GetInstanceCIDs(),
		Timestamp:    p.Timestamp,
	})
}

func (p *Peer) UnmarshalJSON(data []byte) error {
	var temp struct {
		ID           PeerID
		OwnerGUID    InstanceGUID
		ConceptCIDs  []CID
		InstanceCIDs []CID
		Timestamp    time.Time
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	p.ID = temp.ID
	p.OwnerGUID = temp.OwnerGUID
	p.Timestamp = temp.Timestamp
	p.ConceptCIDs = make(map[CID]bool)
	for _, cid := range temp.ConceptCIDs {
		p.ConceptCIDs[cid] = true
	}
	p.InstanceCIDs = make(map[CID]bool)
	for _, cid := range temp.InstanceCIDs {
		p.InstanceCIDs[cid] = true
	}

	return nil
}

type PeerMessage struct {
	PeerID        PeerID
	OwnerGUID     InstanceGUID
	ConceptCIDs   []CID
	InstanceCIDs  []CID
	Relationships RelationshipMap
}

type PeerMap map[PeerID]Peer_i

type ConceptFilter struct {
	CID            CID
	GUID           ConceptGUID
	Name           string
	Description    string
	Type           string
	TimestampAfter *time.Time
}

var (
	conceptMap    ConceptMap
	conceptID2CID ConceptGUID2CIDMap
	conceptMu     sync.RWMutex

	relationshipMap RelationshipMap

	peerMap   PeerMap
	peerMapMu sync.RWMutex
	peerID    PeerID

	ownerGUID InstanceGUID
	ownerMu   sync.RWMutex

	instanceMap    ConceptInstanceMap
	instanceID2CID InstanceGUID2CIDMap
)

func (g ConceptGUID) AsConcept() *Concept {
	c, ok := conceptMap[g]
	if ok {
		return c
	}
	return nil
}

func (cid CID) AsConcept(ctx context.Context) (*Concept, error) {
	conceptReader, err := network.Get(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("unable to get Concept: %s: %v", cid, err)
	}
	var c Concept
	err = json.NewDecoder(conceptReader).Decode(&c)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Concept: %s: %v", cid, err)
	}
	c.CID = cid
	return &c, nil
}

func (cid CID) AsInstanceConcept(ctx context.Context) (ConceptInstance_i, error) {
	conceptReader, err := network.Get(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("unable to get Instance: %s: %v", cid, err)
	}
	data, err := io.ReadAll(conceptReader)
	if err != nil {
		return nil, fmt.Errorf("unable to read data from concept reader: %s: %v", cid, err)
	}

	instance, err := UnmarshalJSON2ConceptInstance(data)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Instance: %s: %v", cid, err)
	}
	instance.SetCID(cid)
	return instance, nil
}

func (pm *PeerMap) UnmarshalJSON(data []byte) error {
	var rawMap map[PeerID]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	*pm = make(PeerMap)
	for peerID, raw := range rawMap {
		var p Peer
		if err := json.Unmarshal(raw, &p); err != nil {
			return err
		}
		(*pm)[peerID] = &p
	}
	return nil
}

func (rm *RelationshipMap) UnmarshalJSON(data []byte) error {
	var rawMap map[ConceptGUID]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	*rm = make(RelationshipMap)
	for guid, raw := range rawMap {
		var r Relationship
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		(*rm)[guid] = &r
	}
	return nil
}
