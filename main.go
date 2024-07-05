package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
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
