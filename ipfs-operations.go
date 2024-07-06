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

func saveInstances(ctx context.Context) error {
	if err := saveData(ctx, instancesPath, instanceMap); err != nil {
		return err
	}
	if err := saveData(ctx, instanceID2CIDPath, instanceID2CID); err != nil {
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

	instanceCIDs := make([]CID, 0, len(instanceMap))
	for _, instance := range instanceMap {
		instanceCIDs = append(instanceCIDs, instance.GetCID())
	}

	message := PeerMessage{
		PeerID:        peerID,
		StewardID:     peer.GetStewardID(),
		ConceptCIDs:   conceptCIDs,
		InstanceCIDs:  instanceCIDs,
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
