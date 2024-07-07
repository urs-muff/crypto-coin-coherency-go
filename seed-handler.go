package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

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
	var seedData map[string]any

	if err := c.BindJSON(&seedData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request body"})
		return
	}

	conceptID, ok := seedData["ConceptID"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ConceptID is required"})
		return
	}

	generator := &SeedNursery{}
	seed, err := generator.CreateSeed(ConceptGUID(conceptID), seedData)
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

func updateSeed_h(c *gin.Context) {
	seedID := SeedGUID(c.Param("guid"))

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request body"})
		return
	}
	updatedSeed, err := UnmarshalJSON2Seed(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request body"})
		return
	}

	existingSeed, exists := seedMap[seedID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seed not found"})
		return
	}

	updatedSeed.SetCID(existingSeed.GetCID())
	updatedSeed.GetCoreSeed().Timestamp = time.Now()
	if err := addOrUpdateSeed(c.Request.Context(), updatedSeed, peerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update seed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"guid": seedID,
		"cid":  string(updatedSeed.GetCID()),
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

func getSteward_h(c *gin.Context) {
	stewardSeed, exists := seedMap[stewardID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Steward not found"})
		return
	}
	c.JSON(http.StatusOK, stewardSeed)
}

func updateSteward_h(c *gin.Context) {
	var stewardSeed StewardSeed
	if err := c.BindJSON(&stewardSeed); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid steward data"})
		return
	}

	existingSteward, exists := seedMap[stewardID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Steward not found"})
		return
	}

	stewardSeed.CID = existingSteward.GetCID()
	stewardSeed.ConceptID = StewardConcept
	stewardSeed.SeedID = stewardID
	// stewardSeed.EnergyBalance = existingSteward.GetSeedID().AsStewardSeed().EnergyBalance
	stewardSeed.Timestamp = time.Now()

	addOrUpdateSeed(c.Request.Context(), &stewardSeed, peerID)

	c.JSON(http.StatusOK, gin.H{"message": "Steward updated successfully", "guid": stewardSeed.SeedID})
}
