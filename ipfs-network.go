package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	peerListPath      = "/ccn/peer-list.json"
	stewardGUIDPath   = "/ccn/steward-guid.json"
	relationshipsPath = "/ccn/relationships.json"

	conceptsPath      = "/ccn/concepts.json"
	conceptID2CIDPath = "/ccn/conceptID-CID.json"

	seedsPath      = "/ccn/seeds.json"
	seedID2CIDPath = "/ccn/seedID-CID.json"
)

func (c *Concept) Update(ctx context.Context) error {
	if c.CID != "" {
		network.Remove(ctx, c.CID)
		c.CID = ""
	}
	conceptJSON, _ := json.Marshal(c)
	cid, err := network.Add(ctx, strings.NewReader(string(conceptJSON)))
	if err != nil {
		return err
	}
	c.CID = cid
	return nil
}

func addOrUpdateConcept(ctx context.Context, concept *Concept, pID PeerID) error {
	conceptMu.Lock()
	defer conceptMu.Unlock()

	if err := concept.Update(ctx); err != nil {
		log.Printf("Failed to update concept: %v", err)
		return err
	}
	conceptMap[concept.GetGUID()] = concept
	conceptID2CID[concept.GetGUID()] = concept.GetCID()
	log.Printf("Added/Updated concept: %s\n", concept)

	if err := saveConcepts(ctx); err != nil {
		log.Printf("Failed to save concept list: %v", err)
		return err
	}

	peerMap[pID].AddConceptCID(concept.GetCID())
	if err := savePeerList(ctx); err != nil {
		log.Printf("Failed to save peer list: %v", err)
	}
	return nil
}

func addNewConcept(ctx context.Context, concept *Concept, pID PeerID) {
	addOrUpdateConcept(ctx, concept, pID)

	go publishPeerMessage(context.Background())
}

func periodicSend(conn *websocket.Conn, sendFunc func(*websocket.Conn)) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		sendFunc(conn)
	}
}

func keepAlive(conn *websocket.Conn) {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
	}
}

func sendConceptList(conn *websocket.Conn) {
	sendJSONList(conn, conceptMap, &conceptMu, "concept map")
}

func sendPeerList(conn *websocket.Conn) {
	peerMapMu.RLock()
	defer peerMapMu.RUnlock()

	filteredPeerMap := make(map[PeerID]Peer_i)
	for peerID, peer := range peerMap {
		if peer.GetStewardID() != "" {
			filteredPeerMap[peerID] = peer
		}
	}

	if err := conn.WriteJSON(filteredPeerMap); err != nil {
		log.Printf("Failed to send peer list: %v", err)
	} else {
		log.Printf("Sent filtered peer list with %d peers", len(filteredPeerMap))
	}
}

func sendJSONList(conn *websocket.Conn, list any, mu sync.Locker, itemType string) {
	mu.Lock()
	defer mu.Unlock()

	if err := conn.WriteJSON(list); err != nil {
		log.Printf("Failed to send %s: %v", itemType, err)
	} else {
		log.Printf("Sent %s", itemType)
	}
}

func handleReceivedMessage(data []byte) {
	var message PeerMessage
	if err := json.Unmarshal(data, &message); err != nil {
		log.Printf("Error unmarshaling received message: %v", err)
		return
	}

	log.Printf("Received message from peer: %s", message.PeerID)

	// Add or update the sender in the peer list
	addOrUpdatePeer(context.Background(), message.PeerID, message.StewardID)

	// Update local relationships with received ones
	for id, relationship := range message.Relationships {
		if _, exists := relationshipMap[id]; !exists {
			relationshipMap[id] = relationship
		}
	}
	saveRelationships(context.Background())

	// Update the CIDs for this peer
	updatePeerCIDs(message.PeerID, message.ConceptCIDs, message.SeedCIDs)
}

// Modify the Interact method of Relationship
func (r *Relationship) Interact(interactionType ConceptGUID) {
	r.Interactions++
	r.Depth = int(math.Log2(float64(r.Interactions))) + 1
	r.LastInteraction = time.Now()

	// Get the interaction type concept
	conceptMu.RLock()
	interactionConcept, ok := conceptMap[interactionType]
	conceptMu.RUnlock()

	if !ok {
		log.Printf("Interaction type %s not found", interactionType)
		return
	}

	// Apply effects based on the interaction type
	switch interactionConcept.Name {
	case "Music":
		r.FrequencySpec = append(r.FrequencySpec, 440.0) // Add A4 note
		r.Amplitude *= 1.05
	case "Meditation":
		r.EnergyFlow *= 1.1
		r.Volume *= 0.95
	case "FlowState":
		r.EnergyFlow *= 1.2
		r.Amplitude *= 1.1
		r.Volume *= 1.05
	default:
		r.EnergyFlow *= 1.05
	}

	r.Timestamp = time.Now()
}
