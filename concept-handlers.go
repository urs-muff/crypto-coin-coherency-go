package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func getConcept_h(c *gin.Context) {
	guid := ConceptGUID(c.Param("guid"))

	conceptMu.RLock()
	concept, exists := conceptMap[guid]
	conceptMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Concept not found"})
		return
	}
	c.JSON(http.StatusOK, concept)
}

func getConceptName_h(c *gin.Context) {
	guid := ConceptGUID(c.Param("guid"))

	conceptMu.RLock()
	concept, exists := conceptMap[guid]
	conceptMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Concept not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"name": concept.Name})
}

func addConcept_h(c *gin.Context) {
	var newConcept struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ConceptType string `json:"type"`
	}

	if err := c.BindJSON(&newConcept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request body"})
		return
	}

	concept := &Concept{
		ID:            ConceptGUID(uuid.New().String()),
		Name:          newConcept.Name,
		Description:   newConcept.Description,
		ConceptType:   newConcept.ConceptType,
		Timestamp:     time.Now(),
		Relationships: []RelationshipGUID{},
	}

	addNewConcept(c.Request.Context(), concept, peerID)
	c.JSON(http.StatusOK, gin.H{
		"guid": concept.ID,
		"cid":  string(concept.CID),
	})
}

func updateConcept_h(c *gin.Context) {
	// Get the concept ID from the URL parameters
	conceptID := ConceptGUID(c.Param("guid"))

	// Struct to hold the updated concept data
	var updatedConcept struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}

	// Bind the JSON body to the updatedConcept struct
	if err := c.BindJSON(&updatedConcept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request body"})
		return
	}

	// Find the existing concept
	conceptMu.Lock()
	existingConcept, exists := conceptMap[conceptID]
	conceptMu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Concept not found"})
		return
	}

	// Update the concept fields
	existingConcept.Name = updatedConcept.Name
	existingConcept.Description = updatedConcept.Description
	existingConcept.ConceptType = updatedConcept.Type
	existingConcept.Timestamp = time.Now()

	// Use the existing function to update the concept
	err := addOrUpdateConcept(c.Request.Context(), existingConcept, peerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update concept"})
		return
	}

	// Return the updated concept
	c.JSON(http.StatusOK, gin.H{
		"guid": existingConcept.ID,
		"cid":  string(existingConcept.CID),
	})
}

func deleteConcept_h(c *gin.Context) {
	guid := ConceptGUID(c.Param("guid"))

	conceptMu.Lock()
	defer conceptMu.Unlock()

	concept, exists := conceptMap[guid]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Concept not found"})
		return
	}

	if err := network.Remove(c.Request.Context(), concept.GetCID()); err != nil {
		log.Printf("Failed to remove concept: %v", err)
	}
	delete(conceptMap, guid)
	delete(conceptID2CID, guid)
	if err := saveConcepts(c.Request.Context()); err != nil {
		log.Printf("Failed to save concept map: %v", err)
	}

	c.Status(http.StatusNoContent)
}

func queryConcepts_h(c *gin.Context) {
	filter := ConceptFilter{
		CID:         CID(c.Query("cid")),
		GUID:        ConceptGUID(c.Query("guid")),
		Name:        c.Query("name"),
		Description: c.Query("description"),
		Type:        c.Query("type"),
	}

	if timestamp := c.Query("timestamp"); timestamp != "" {
		t, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format"})
			return
		}
		filter.TimestampAfter = &t
	}

	concepts := filterConcepts(filter)
	c.JSON(http.StatusOK, concepts)
}
