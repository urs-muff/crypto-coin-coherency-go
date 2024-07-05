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

// GUID represents a globally unique identifier
type GUID string

type GUID2CIDMap map[GUID]CID

// Peer_i represents a peer in the network
type Peer_i interface {
	GetID() PeerID
	GetOwnerGUID() GUID
	GetCIDs() []CID
	GetTimestamp() time.Time
	AddCID(cid CID)
	RemoveCID(cid CID)
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
	GUID          GUID
	Name          string
	Description   string
	Type          string
	Relationships []GUID
	Timestamp     time.Time
}

type ConceptMap map[GUID]*Concept

func (c Concept) GetCID() CID              { return c.CID }
func (c Concept) GetGUID() GUID            { return c.GUID }
func (c Concept) GetName() string          { return c.Name }
func (c Concept) GetDescription() string   { return c.Description }
func (c Concept) GetType() string          { return c.Type }
func (c Concept) GetRelationships() []GUID { return c.Relationships }
func (c Concept) GetTimestamp() time.Time  { return c.Timestamp }
func (c Concept) String() string           { return fmt.Sprintf("%s: %s (%s)", c.GUID, c.Name, c.Type) }

// Relationship represents a connection between two entities
type Relationship struct {
	ID              GUID
	SourceID        GUID
	TargetID        GUID
	Type            GUID
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
type RelationshipMap map[GUID]*Relationship

// Function to create a new relationship
func CreateRelationship(sourceID, targetID GUID, relationType GUID) *Relationship {
	return &Relationship{
		ID:            GUID(uuid.New().String()),
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
	ID        PeerID
	OwnerGUID GUID
	CIDs      map[CID]bool
	Timestamp time.Time
}

func (p Peer) GetID() PeerID      { return p.ID }
func (p Peer) GetOwnerGUID() GUID { return p.OwnerGUID }
func (p Peer) GetCIDs() []CID {
	ret := make([]CID, 0)
	for cid := range p.CIDs {
		ret = append(ret, cid)
	}
	return ret
}
func (p Peer) GetTimestamp() time.Time { return p.Timestamp }
func (p *Peer) AddCID(cid CID)         { p.CIDs[cid] = true }
func (p *Peer) RemoveCID(cid CID)      { delete(p.CIDs, cid) }

func (p *Peer) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        PeerID
		OwnerGUID GUID
		CIDs      []CID
		Timestamp time.Time
	}{
		ID:        p.ID,
		OwnerGUID: p.OwnerGUID,
		CIDs:      p.GetCIDs(),
		Timestamp: p.Timestamp,
	})
}

func (p *Peer) UnmarshalJSON(data []byte) error {
	var temp struct {
		ID        PeerID    `json:"id"`
		OwnerGUID GUID      `json:"ownerGuid"`
		CIDs      []CID     `json:"cids"`
		Timestamp time.Time `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	p.ID = temp.ID
	p.OwnerGUID = temp.OwnerGUID
	p.Timestamp = temp.Timestamp
	p.CIDs = make(map[CID]bool)

	for _, cid := range temp.CIDs {
		p.CIDs[cid] = true
	}

	return nil
}

type PeerMessage struct {
	PeerID        PeerID          `json:"peerId"`
	OwnerGUID     GUID            `json:"ownerGuid"`
	CIDs          []CID           `json:"cids"`
	Relationships RelationshipMap `json:"relationships"`
}

type PeerMap map[PeerID]Peer_i

type ConceptFilter struct {
	CID            string
	GUID           GUID
	Name           string
	Description    string
	Type           string
	TimestampAfter *time.Time
}

var (
	conceptMap ConceptMap
	GUID2CID   GUID2CIDMap
	conceptMu  sync.RWMutex

	relationshipMap RelationshipMap

	peerMap   PeerMap
	peerMapMu sync.RWMutex
	peerID    PeerID

	ownerGUID GUID
	ownerMu   sync.RWMutex
)

func (g GUID) AsConcept() *Concept {
	c, ok := conceptMap[g]
	if ok {
		return c
	}
	return nil
}
