package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getSeed_h(c *gin.Context) {
	guid := SeedGUID(c.Param("guid"))

	seed, exists := seedMap[guid]

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seed not found"})
		return
	}
	c.JSON(http.StatusOK, seed)
}

func addSeed_h(c *gin.Context) {
	var newSeed struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Concept     string         `json:"concept"`
		Data        map[string]any `json:"data"`
	}

	if err := c.BindJSON(&newSeed); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request body"})
		return
	}

	generator := &SeedNursery{}
	seed, err := generator.CreateSeed(newSeed.Concept, newSeed.Name, newSeed.Description, newSeed.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to create seed: %v", err)})
		return
	}

	if err := addOrUpdateSeed(c.Request.Context(), seed, peerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add seed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"guid": seed.GetSeedID(),
		"cid":  string(seed.GetCID()),
	})
}

func deleteSeed_h(c *gin.Context) {
	guid := SeedGUID(c.Param("guid"))

	seed, exists := seedMap[guid]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seed not found"})
		return
	}

	if err := network.Remove(c.Request.Context(), seed.GetCID()); err != nil {
		log.Printf("Failed to remove seed: %v", err)
	}
	delete(seedMap, guid)
	delete(seedID2CID, guid)
	if err := saveSeeds(c.Request.Context()); err != nil {
		log.Printf("Failed to save seed map: %v", err)
	}

	c.Status(http.StatusNoContent)
}

func querySeeds_h(c *gin.Context) {
	//	filter := SeedFilter{
	//		CID:         CID(c.Query("cid")),
	//		GUID:        SeedGUID(c.Query("guid")),
	//		Name:        c.Query("name"),
	//		Description: c.Query("description"),
	//		Type:        c.Query("type"),
	//	}
	//
	//	if timestamp := c.Query("timestamp"); timestamp != "" {
	//		t, err := time.Parse(time.RFC3339, timestamp)
	//		if err != nil {
	//			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format"})
	//			return
	//		}
	//		filter.TimestampAfter = &t
	//	}
	//
	//	seeds := filterSeeds(filter)
	seeds := []Seed_i{}
	for _, seed := range seedMap {
		seeds = append(seeds, seed)
	}
	c.JSON(http.StatusOK, seeds)
}
