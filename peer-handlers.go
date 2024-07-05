package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func listPeers_h(c *gin.Context) {
	peerMapMu.RLock()
	defer peerMapMu.RUnlock()

	filteredPeerMap := make(map[PeerID]Peer_i)
	for peerID, peer := range peerMap {
		if peer.GetOwnerGUID() != "" {
			filteredPeerMap[peerID] = peer
		}
	}

	c.JSON(http.StatusOK, filteredPeerMap)
}

func addOrUpdatePeer(ctx context.Context, peerID PeerID, ownerGUID GUID) {
	peerMapMu.Lock()
	defer peerMapMu.Unlock()

	peerMap[peerID] = &Peer{
		ID:        peerID,
		OwnerGUID: ownerGUID,
		Timestamp: time.Now(),
	}
	log.Printf("Updated peer: %s", peerID)

	if err := savePeerList(ctx); err != nil {
		log.Printf("Failed to save peerMap: %v", err)
	}
}

func updatePeerCIDs(peerID PeerID, cids []CID) {
	conceptMu.Lock()
	defer conceptMu.Unlock()

	for _, cid := range cids {
		found := false
		for _, concept := range conceptMap {
			if concept.GetCID() == cid {
				found = true
				break
			}
		}
		if !found {
			// If the concept is not in our list, we might want to fetch it
			// This is left as an exercise, as it depends on how you want to handle this case
			log.Printf("Found new CID from peer %s: %s", peerID, cid)
		}
	}
}

func discoverPeers(ctx context.Context) {
	//	peers, err := network.ListPeers(ctx)
	//	if err != nil {
	//		log.Printf("Error discovering peers: %v", err)
	//		return
	//	}
	//
	//	peerMapMu.Lock()
	//	defer peerMapMu.Unlock()
	//
	//	for _, peer := range peers {
	//		peerID := peer.GetID()
	//		if _, exists := peerMap[peerID]; !exists {
	//			peerMap[peerID] = peer
	//			log.Printf("Discovered new peer: %s", peerID)
	//		}
	//	}
	//
	//	log.Printf("Discovered %d peers", len(peerMap))
	//
	//	if err := network.Save(context.Background(), savePeerList(ctx), peerMap); err != nil {
	//		log.Printf("Failed to save peerMap: %v", err)
	//	}
}
