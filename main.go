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
	r.DELETE("/concept/:guid", deleteConcept_h)
	r.PUT("/concept/:guid", updateConcept_h)
	r.GET("/concept/:guid", getConcept_h)
	r.GET("/concept/:guid/name", getConceptName_h)
	r.GET("/concepts", queryConcepts_h)

	r.PUT("/steward", updateSteward_h)
	r.GET("/steward", getSteward_h)

	r.POST("/seed", addSeed_h)
	r.DELETE("/seed/:guid", deleteSeed_h)
	r.PUT("/seed/:guid", updateSeed_h)
	r.GET("/seed/:guid", getSeed_h)
	r.GET("/seeds", querySeeds_h)

	r.GET("/peers", listPeers_h)

	r.GET("/ws", handleWebSocket_h)
	r.GET("/ws/peers", handlePeerWebSocket_h)

	r.POST("/relationship", addRelationship_h)
	r.PUT("/relationship/:id/deepen", deepenRelationship_h)
	r.GET("/relationships", getRelationships_h)
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
	seedMap = make(SeedMap)
	seedID2CID = make(SeedGUID2CIDMap)

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
		if peer.GetStewardID() == "" {
			delete(peerMap, id)
		}
	}
	if _, ok := peerMap[peerID]; !ok {
		peerMap[peerID] = &Peer{
			ID:          peerID,
			Timestamp:   time.Now(),
			ConceptCIDs: make(map[CID]bool),
			SeedCIDs:    make(map[CID]bool),
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
			guidMap[concept.Name] = GUID(id)
		}
	}
	StewardConcept = findConceptGUID("Steward")
	AssetConcept = findConceptGUID("Asset")
	CoinConcept = findConceptGUID("Coin")
	SmartContractConcept = findConceptGUID("Smart Contract")
	ContractEvaluatorConcept = findConceptGUID("Contract Evaluator")
	ConceptInvestmentConcept = findConceptGUID("Concept Investment")
	SeedInvestmentConcept = findConceptGUID("Seed Investment")
	TransactionConcept = findConceptGUID("Transaction")
	ReturnConcept = findConceptGUID("Return")
	ProposalConcept = findConceptGUID("Proposal")
	ProposalActionConcept = findConceptGUID("Proposal Action")
	HarmonyGuidelineConcept = findConceptGUID("Harmony Guideline")
	initSeedUnmarshal()

	if err := network.Load(ctx, seedID2CIDPath, &seedID2CID); err != nil {
		log.Printf("Failed to load seed CID map: %v\n", err)
	}
	if err := network.Load(ctx, seedsPath, &seedMap); err != nil {
		log.Printf("Failed to load seeds: %v\n", err)
	}

	loadOrCreateSteward(ctx)
	peerMap[peerID].(*Peer).StewardID = stewardID
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

	for _, cid := range peerMap[peerID].GetSeedCIDs() {
		i, err := cid.AsSeed(ctx)
		if err != nil {
			log.Fatalf("Unable to parse seed: %s: %v", cid, err)
		} else {
			log.Printf("Seed: %s\n", i)
		}
	}
}

func loadOrCreateSteward(ctx context.Context) {
	var guid SeedGUID
	err := network.Load(ctx, stewardGUIDPath, &guid)
	if err != nil {
		log.Printf("Failed to load Steward ID from IPFS: %v", err)
		log.Println("Generating new Steward ID...")
		guid = SeedGUID(uuid.New().String())
		if err := network.Save(ctx, stewardGUIDPath, guid); err != nil {
			log.Fatalf("Failed to save new Steward ID: %v", err)
		}
	}

	stewardMu.Lock()
	stewardID = guid
	stewardMu.Unlock()

	log.Printf("Steward ID: %s", stewardID)
	_, ok := seedID2CID[stewardID]
	if !ok {
		steward := NewStewardSeed("Urs Muff", "Creator of this network")
		steward.SeedID = stewardID
		addOrUpdateSeed(ctx, steward, peerID)

		// if err := network.Load(ctx, seedsPath, &seedMap); err != nil {
		// 	log.Printf("Failed to load seeds: %v\n", err)
		// }

		// peerJson, _ := json.Marshal(peerMap[peerID])
		// fmt.Printf("Peer: %s\n", string(peerJson))

		// cid := seedID2CID[stewardID]
		//
		// if seed, err := cid.AsSeed(ctx); err != nil {
		// 	log.Fatalf("Unable to parse Concept: %s: %v", cid, err)
		// } else {
		// 	log.Printf("Seed: %s\n", seed)
		// }
	}
}
