package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
)

func initializeLists(ctx context.Context) {
	conceptMap = make(ConceptMap)
	GUID2CID = make(GUID2CIDMap)
	peerMap = make(PeerMap)
	relationshipMap = make(RelationshipMap)

	if err := network.Bootstrap(ctx); err != nil {
		log.Fatalf("Failed to bootstrap IPFS: %v", err)
	}

	var err error
	peerID, err = network.ID(ctx)
	if err != nil {
		log.Fatalf("Failed to get peer ID: %v", err)
	}

	if err := network.Load(ctx, GUID2CIDPath, &GUID2CID); err != nil {
		log.Printf("Failed to load concept list: %v\n", err)
	}
	if err := network.Load(ctx, relationshipsPath, &relationshipMap); err != nil {
		log.Printf("Failed to load relationships: %v", err)
		InitializeSystem(ctx)
		saveRelationships(ctx)
	}
	if err := network.Load(ctx, peerListPath, &peerMap); err != nil {
		log.Printf("Failed to load peer list: %v\n", err)
	}
	for id, peer := range peerMap {
		if peer.GetOwnerGUID() == "" {
			delete(peerMap, id)
		}
	}
	peerMap[peerID] = &Peer{
		ID:        peerID,
		Timestamp: time.Now(),
		CIDs:      make(map[CID]bool),
	}
	loadOrCreateOwner(ctx)
	peerMap[peerID].(*Peer).OwnerGUID = ownerGUID
	savePeerList(ctx)

	for _, cid := range peerMap[peerID].GetCIDs() {
		conceptReader, err := network.Get(context.Background(), cid)
		if err != nil {
			log.Fatalf("Unable to get Concept: %s: %v", cid, err)
		}
		var c Concept
		err = json.NewDecoder(conceptReader).Decode(&c)
		if err != nil {
			log.Fatalf("Unable to parse Concept: %s: %v", cid, err)
		}
		c.CID = cid
		conceptMap[c.GUID] = &c
		GUID2CID[c.GUID] = cid
	}
}

func loadOrCreateOwner(ctx context.Context) {
	var guid GUID
	err := network.Load(ctx, ownerGUIDPath, &guid)
	if err != nil {
		log.Printf("Failed to load owner GUID from IPFS: %v", err)
		log.Println("Generating new owner GUID...")
		guid = GUID(uuid.New().String())
		if err := network.Save(ctx, ownerGUIDPath, guid); err != nil {
			log.Fatalf("Failed to save new owner GUID: %v", err)
		}
	}

	ownerMu.Lock()
	ownerGUID = guid
	ownerMu.Unlock()

	log.Printf("Owner GUID: %s", ownerGUID)
	cid, ok := GUID2CID[ownerGUID]
	if !ok {
		ownerConcept := &Concept{
			GUID:          guid,
			Name:          "Owner",
			Description:   "Owner",
			Type:          "Owner",
			Timestamp:     time.Now(),
			Relationships: []GUID{},
		}
		addOrUpdateConcept(context.Background(), ownerConcept)
		cid = ownerConcept.CID
	}
	peerMap[peerID].AddCID(cid)
}

func saveRelationships(ctx context.Context) error {
	if err := network.Save(ctx, relationshipsPath, relationshipMap); err != nil {
		log.Printf("Failed to save relationships: %v", err)
		return err
	}
	return nil
}

func saveConceptMap(ctx context.Context) error {
	if err := network.Save(ctx, conceptsPath, conceptMap); err != nil {
		log.Printf("Failed to save concept map: %v", err)
		return err
	}
	return nil
}

func savePeerList(ctx context.Context) error {
	if err := network.Save(ctx, peerListPath, peerMap); err != nil {
		log.Printf("Failed to save peer list: %v", err)
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
	cids := make([]CID, 0, len(conceptMap))
	for _, concept := range conceptMap {
		cids = append(cids, concept.GetCID())
	}
	conceptMu.RUnlock()

	message := PeerMessage{
		PeerID:        peerID,
		OwnerGUID:     peer.GetOwnerGUID(),
		CIDs:          cids,
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
		log.Printf("Published peer message with %d CIDs", len(cids))
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
