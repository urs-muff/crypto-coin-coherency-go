package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func addRelationship_h(c *gin.Context) {
	var req struct {
		SourceID EntityGUID  `json:"sourceId"`
		TargetID EntityGUID  `json:"targetId"`
		TypeID   ConceptGUID `json:"typeId"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	relationship := CreateRelationship(req.SourceID, req.TargetID, req.TypeID, map[string]any{})
	relationshipMu.Lock()
	relationshipMap[relationship.ID] = relationship
	relationshipMu.Unlock()

	// Update the concepts
	conceptMu.Lock()
	if concept, ok := conceptMap[ConceptGUID(req.SourceID)]; ok {
		concept.Relationships = append(concept.Relationships, relationship.ID)
	}
	if concept, ok := conceptMap[ConceptGUID(req.TargetID)]; ok {
		concept.Relationships = append(concept.Relationships, relationship.ID)
	}
	conceptMu.Unlock()

	// Save updated data
	saveRelationships(c.Request.Context())
	saveConcepts(c.Request.Context())

	c.JSON(http.StatusOK, relationship)
}

func deepenRelationship_h(c *gin.Context) {
	id := RelationshipGUID(c.Param("id"))
	if relationship, ok := relationshipMap[id]; ok {
		// relationship.Deepen()
		relationshipMu.Lock()
		relationshipMap[id] = relationship
		relationshipMu.Unlock()
		saveRelationships(c.Request.Context())
		c.JSON(http.StatusOK, relationship)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Relationship not found"})
	}
}

func getRelationships_h(c *gin.Context) {
	relationships := []Relationship{}
	relationshipMu.RLock()
	defer relationshipMu.RUnlock()
	for _, relationship := range relationshipMap {
		relationships = append(relationships, *relationship)
	}

	c.JSON(http.StatusOK, relationships)
}

func getRelationship_h(c *gin.Context) {
	id := RelationshipGUID(c.Param("id"))
	relationshipMu.RLock()
	defer relationshipMu.RUnlock()
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
		if concept.ConceptType == "RelationshipType" {
			relationshipTypes = append(relationshipTypes, *concept)
		}
	}

	c.JSON(http.StatusOK, relationshipTypes)
}

func getRelationshipsByType_h(c *gin.Context) {
	typeGUID := ConceptGUID(c.Query("type"))
	var filteredRelationships []*Relationship
	for _, rel := range relationshipMap {
		if rel.Type == typeGUID {
			filteredRelationships = append(filteredRelationships, rel)
		}
	}
	c.JSON(http.StatusOK, filteredRelationships)
}

func interactWithRelationship_h(c *gin.Context) {
	id := RelationshipGUID(c.Param("id"))
	var req struct {
		InteractionTypeGUID ConceptGUID `json:"interactionTypeGuid"`
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
