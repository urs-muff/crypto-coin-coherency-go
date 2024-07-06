package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func addConcept_h(c *gin.Context) {
	var newConcept struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}

	if err := c.BindJSON(&newConcept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request body"})
		return
	}

	concept := &Concept{
		ID:            ConceptGUID(uuid.New().String()),
		Name:          newConcept.Name,
		Description:   newConcept.Description,
		Type:          newConcept.Type,
		Timestamp:     time.Now(),
		Relationships: []ConceptGUID{},
	}

	addNewConcept(c.Request.Context(), concept, peerID)
	c.JSON(http.StatusOK, gin.H{
		"guid": concept.ID,
		"cid":  string(concept.CID),
	})
}

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

func updateSteward_h(c *gin.Context) {
	var stewardInstance StewardInstance
	if err := c.BindJSON(&stewardInstance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid steward data"})
		return
	}

	stewardInstance.ConceptID = StewardConcept
	stewardInstance.InstanceID = stewardID
	stewardInstance.Timestamp = time.Now()

	addOrUpdateInstance(c.Request.Context(), &stewardInstance, peerID)

	c.JSON(http.StatusOK, gin.H{"message": "Steward updated successfully", "guid": stewardInstance.InstanceID})
}

func getSteward_h(c *gin.Context) {
	stewardInstance, exists := instanceMap[stewardID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Steward not found"})
		return
	}
	c.JSON(http.StatusOK, stewardInstance)
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
