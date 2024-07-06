package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	pubsubTopic       = "concept-list"
	publishInterval   = 1 * time.Minute
	peerCheckInterval = 5 * time.Minute
)

var (
	network Node_i
)

func main() {
	network = NewIPFSShell("localhost:5001")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initializeLists(ctx)

	// Start IPFS routines
	go runPeriodicTask(ctx, publishInterval, publishPeerMessage)
	go runPeriodicTask(ctx, peerCheckInterval, discoverPeers)
	go subscribeRoutine(ctx)

	// Set up Gin router
	r := gin.Default()
	setupRoutes(r)

	// Start server
	log.Fatal(r.Run(":9090"))
}

func setupRoutes(r *gin.Engine) {
	r.Use(corsMiddleware())
	r.POST("/concept", addConcept_h)
	r.GET("/concept/:guid", getConcept_h)
	r.POST("/owner", updateOwner_h)
	r.GET("/owner", getOwner_h)
	r.DELETE("/concept/:guid", deleteConcept_h)
	r.GET("/concepts", queryConcepts_h)
	r.GET("/peers", listPeers_h)
	r.GET("/ws", handleWebSocket_h)
	r.GET("/ws/peers", handlePeerWebSocket_h)
	r.POST("/relationship", addRelationship_h)
	r.PUT("/relationship/:id/deepen", deepenRelationship_h)
	r.GET("/relationship/:id", getRelationship_h)
	r.GET("/relationship-types", getRelationshipTypes_h)
	r.GET("/relationship-type/:type", getRelationshipsByType_h)
	r.GET("/interact/:id", interactWithRelationship_h)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func runPeriodicTask(ctx context.Context, interval time.Duration, task func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			task(ctx)
		}
	}
}

func initializeLists(ctx context.Context) {
	conceptMap = make(ConceptMap)
	conceptID2CID = make(ConceptGUID2CIDMap)
	peerMap = make(PeerMap)
	relationshipMap = make(RelationshipMap)
	instanceMap = make(ConceptInstanceMap)
	instanceID2CID = make(InstanceGUID2CIDMap)

	if err := network.Bootstrap(ctx); err != nil {
		log.Fatalf("Failed to bootstrap IPFS: %v", err)
	}

	var err error
	peerID, err = network.ID(ctx)
	if err != nil {
		log.Fatalf("Failed to get peer ID: %v", err)
	}
	if err := network.Load(ctx, peerListPath, &peerMap); err != nil {
		log.Printf("Failed to load peer list: %v\n", err)
	}
	for id, peer := range peerMap {
		if peer.GetOwnerGUID() == "" {
			delete(peerMap, id)
		}
	}
	if _, ok := peerMap[peerID]; !ok {
		peerMap[peerID] = &Peer{
			ID:           peerID,
			Timestamp:    time.Now(),
			ConceptCIDs:  make(map[CID]bool),
			InstanceCIDs: make(map[CID]bool),
		}
	}

	if err := network.Load(ctx, conceptID2CIDPath, &conceptID2CID); err != nil {
		log.Printf("Failed to load concept CID map: %v\n", err)
	}
	if err := network.Load(ctx, relationshipsPath, &relationshipMap); err != nil {
		log.Printf("Failed to load relationships: %v\n", err)
		InitializeSystem(ctx)
		saveRelationships(ctx)
	} else {
		if err := network.Load(ctx, conceptsPath, &conceptMap); err != nil {
			log.Fatalf("Failed to load concepts: %v", err)
		}
		for id, concept := range conceptMap {
			guidMap[concept.Name] = id
		}
	}
	OwnerConcept = findGUID("Owner")
	AssetConcept = findGUID("Asset")
	CoinConcept = findGUID("Coin")
	SmartContractConcept = findGUID("Smart Contract")
	ContractEvaluatorConcept = findGUID("Contract Evaluator")
	InvestmentConcept = findGUID("Investment")
	TransactionConcept = findGUID("Transaction")
	ReturnConcept = findGUID("Return")
	initInstanceUnmarshal()

	if err := network.Load(ctx, instanceID2CIDPath, &instanceID2CID); err != nil {
		log.Printf("Failed to load instance CID map: %v\n", err)
	}
	if err := network.Load(ctx, instancesPath, &instanceMap); err != nil {
		log.Printf("Failed to load instance: %v\n", err)
	}

	loadOrCreateOwner(ctx)
	peerMap[peerID].(*Peer).OwnerGUID = ownerGUID
	savePeerList(ctx)

	json, _ := json.Marshal(peerMap[peerID])
	log.Printf("Peer[%s]: %s\n", peerID, string(json))

	for _, cid := range peerMap[peerID].GetConceptCIDs() {
		c, err := cid.AsConcept(ctx)
		if err != nil {
			log.Fatalf("Unable to parse Concept: %s: %v", cid, err)
		} else {
			log.Printf("Concept: %s\n", c)
		}
	}

	for _, cid := range peerMap[peerID].GetInstanceCIDs() {
		i, err := cid.AsInstanceConcept(ctx)
		if err != nil {
			log.Fatalf("Unable to parse Instance: %s: %v", cid, err)
		} else {
			log.Printf("Instance: %s\n", i)
		}
	}
}

func loadOrCreateOwner(ctx context.Context) {
	var guid InstanceGUID
	err := network.Load(ctx, ownerGUIDPath, &guid)
	if err != nil {
		log.Printf("Failed to load owner GUID from IPFS: %v", err)
		log.Println("Generating new owner GUID...")
		guid = InstanceGUID(uuid.New().String())
		if err := network.Save(ctx, ownerGUIDPath, guid); err != nil {
			log.Fatalf("Failed to save new owner GUID: %v", err)
		}
	}

	ownerMu.Lock()
	ownerGUID = guid
	ownerMu.Unlock()

	log.Printf("Owner GUID: %s", ownerGUID)
	_, ok := instanceID2CID[ownerGUID]
	if !ok {
		owner := NewOwnerInstance("Urs Muff", "Creator of this network")
		owner.InstanceID = ownerGUID
		addOrUpdateInstance(ctx, owner, peerID)
		asset1 := NewAssetInstance("First Thing", "", owner.InstanceID)
		addOrUpdateInstance(ctx, asset1, peerID)
		owner.OwnedAssets = append(owner.OwnedAssets, asset1.InstanceID)
		addOrUpdateInstance(ctx, owner, peerID)
		if err := network.Load(ctx, instancesPath, &instanceMap); err != nil {
			log.Printf("Failed to load instance: %v\n", err)
		}

		// peerJson, _ := json.Marshal(peerMap[peerID])
		// fmt.Printf("Peer: %s\n", string(peerJson))

		// cid := instanceID2CID[ownerGUID]
		//
		// if instance, err := cid.AsInstanceConcept(ctx); err != nil {
		// 	log.Fatalf("Unable to parse Concept: %s: %v", cid, err)
		// } else {
		// 	log.Printf("Instance: %s\n", instance)
		// }
	}
}
