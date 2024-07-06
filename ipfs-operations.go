package main

import (
	"context"
	"encoding/json"
	"log"
)

func saveData(ctx context.Context, path string, data any) error {
	if err := network.Save(ctx, path, data); err != nil {
		log.Printf("Failed to save data at %s: %v", path, err)
		return err
	}
	return nil
}

func saveRelationships(ctx context.Context) error {
	return saveData(ctx, relationshipsPath, relationshipMap)
}

func saveConcepts(ctx context.Context) error {
	if err := saveData(ctx, conceptsPath, conceptMap); err != nil {
		return err
	}
	if err := saveData(ctx, conceptID2CIDPath, conceptID2CID); err != nil {
		return err
	}
	return nil
}

func savePeerList(ctx context.Context) error {
	return saveData(ctx, peerListPath, peerMap)
}

func saveSeeds(ctx context.Context) error {
	if err := saveData(ctx, seedsPath, seedMap); err != nil {
		return err
	}
	if err := saveData(ctx, seedID2CIDPath, seedID2CID); err != nil {
		return err
	}
	return nil
}

func publishPeerMessage(ctx context.Context) {
	peerMapMu.RLock()
	peer, exists := peerMap[peerID]
	peerMapMu.RUnlock()

	if !exists {
		log.Printf("Peer information not set for this peer")
		return
	}

	conceptMu.RLock()
	conceptCIDs := make([]CID, 0, len(conceptMap))
	for _, concept := range conceptMap {
		conceptCIDs = append(conceptCIDs, concept.GetCID())
	}
	conceptMu.RUnlock()

	seedCIDs := make([]CID, 0, len(seedMap))
	for _, seed := range seedMap {
		seedCIDs = append(seedCIDs, seed.GetCID())
	}

	message := PeerMessage{
		PeerID:        peerID,
		StewardID:     peer.GetStewardID(),
		ConceptCIDs:   conceptCIDs,
		SeedCIDs:      seedCIDs,
		Relationships: relationshipMap,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling peer message: %v", err)
		return
	}

	if err := network.Publish(ctx, pubsubTopic, data); err != nil {
		log.Printf("Error publishing peer message: %v", err)
	} else {
		log.Printf("Published peer message with %d CIDs", len(conceptCIDs))
	}
}

func subscribeRoutine(ctx context.Context) {
	ch, err := network.Subscribe(ctx, pubsubTopic)
	if err != nil {
		log.Fatalf("Error subscribing to topic: %v", err)
	}

	log.Printf("Subscribed to topic: %s", pubsubTopic)

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			handleReceivedMessage(msg)
		}
	}
}
