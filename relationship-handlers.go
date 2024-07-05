package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func addRelationship_h(c *gin.Context) {
	var req struct {
		SourceID GUID `json:"sourceId"`
		TargetID GUID `json:"targetId"`
		TypeID   GUID `json:"typeId"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	relationship := CreateRelationship(req.SourceID, req.TargetID, req.TypeID)
	relationshipMap[relationship.ID] = relationship

	// Update the concepts
	conceptMu.Lock()
	if concept, ok := conceptMap[req.SourceID]; ok {
		concept.Relationships = append(concept.Relationships, relationship.ID)
	}
	if concept, ok := conceptMap[req.TargetID]; ok {
		concept.Relationships = append(concept.Relationships, relationship.ID)
	}
	conceptMu.Unlock()

	// Save updated data
	saveRelationships(c.Request.Context())
	saveConceptMap(c.Request.Context())

	c.JSON(http.StatusOK, relationship)
}

func deepenRelationship_h(c *gin.Context) {
	id := GUID(c.Param("id"))
	if relationship, ok := relationshipMap[id]; ok {
		relationship.Deepen()
		relationshipMap[id] = relationship
		saveRelationships(c.Request.Context())
		c.JSON(http.StatusOK, relationship)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Relationship not found"})
	}
}

func getRelationship_h(c *gin.Context) {
	id := GUID(c.Param("id"))
	if relationship, ok := relationshipMap[id]; ok {
		c.JSON(http.StatusOK, relationship)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Relationship not found"})
	}
}

func getRelationshipTypes_h(c *gin.Context) {
	conceptMu.RLock()
	defer conceptMu.RUnlock()

	var relationshipTypes []Concept
	for _, concept := range conceptMap {
		if concept.Type == "RelationshipType" {
			relationshipTypes = append(relationshipTypes, *concept)
		}
	}

	c.JSON(http.StatusOK, relationshipTypes)
}

func getRelationshipsByType_h(c *gin.Context) {
	typeGUID := GUID(c.Query("type"))
	var filteredRelationships []*Relationship
	for _, rel := range relationshipMap {
		if rel.Type == typeGUID {
			filteredRelationships = append(filteredRelationships, rel)
		}
	}
	c.JSON(http.StatusOK, filteredRelationships)
}

func interactWithRelationship_h(c *gin.Context) {
	id := GUID(c.Param("id"))
	var req struct {
		InteractionTypeGUID GUID `json:"interactionTypeGuid"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if relationship, ok := relationshipMap[id]; ok {
		relationship.Interact(req.InteractionTypeGUID)
		saveRelationships(c.Request.Context())
		c.JSON(http.StatusOK, relationship)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Relationship not found"})
	}
}
