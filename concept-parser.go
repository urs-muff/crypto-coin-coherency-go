package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type ConceptStructure struct {
	Concepts      []ConceptNode      `yaml:"concepts"`
	Relationships []RelationshipNode `yaml:"relationships"`
}

type ConceptNode struct {
	Name          string             `yaml:"name"`
	Description   string             `yaml:"description"`
	Type          string             `yaml:"type"`
	Children      []ConceptNode      `yaml:"children,omitempty"`
	Relationships []RelationshipType `yaml:"relationships,omitempty"`
}

type RelationshipType struct {
	Type   string `yaml:"type"`
	Target string `yaml:"target"`
}

type RelationshipNode struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

var guidMap = make(map[string]ConceptGUID)

func generateGUID(ctx context.Context, name string) ConceptGUID {
	if guid, exists := guidMap[name]; exists {
		return guid
	}
	guid, err := network.Add(ctx, strings.NewReader(name))
	if err != nil {
		log.Fatalf("Failed to generate GUID: %v", err)
	}
	guidMap[name] = ConceptGUID(guid)
	return ConceptGUID(guid)
}

func findGUID(name string) ConceptGUID {
	if guid, exists := guidMap[name]; exists {
		return guid
	}
	log.Fatalf("GUID for Concept '%s' not found.", name)
	return ""
}

func (guid ConceptGUID) findName() string {
	for name, nameGuid := range guidMap {
		if guid == nameGuid {
			return name
		}
	}
	return ""
}

func parseConceptStructure(filename string) (*ConceptStructure, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var structure ConceptStructure
	err = yaml.Unmarshal(data, &structure)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	return &structure, nil
}

func createConcepts(ctx context.Context, node ConceptNode, parentGUID ConceptGUID) (*Concept, error) {
	guid := generateGUID(ctx, node.Name)
	concept := &Concept{
		ID:          guid,
		Name:        node.Name,
		Description: node.Description,
		Type:        node.Type,
		Timestamp:   time.Now(),
	}

	if len(node.Children) > 0 || parentGUID != "" {
		err := addOrUpdateConcept(ctx, concept, peerID)
		if err != nil {
			return nil, fmt.Errorf("failed to add or update concept %s: %v", node.Name, err)
		}
	}

	if parentGUID != "" {
		err := createCoreRelationship(ctx, parentGUID, generateGUID(ctx, "Component Of"), guid)
		if err != nil {
			return nil, fmt.Errorf("failed to create 'Component Of' relationship for %s: %v", node.Name, err)
		}
	}

	for _, child := range node.Children {
		childConcept, err := createConcepts(ctx, child, guid)
		if err != nil {
			return nil, err
		}
		concept.Relationships = append(concept.Relationships, childConcept.ID)
	}

	err := addOrUpdateConcept(ctx, concept, peerID)
	if err != nil {
		return nil, fmt.Errorf("failed to add or update concept %s: %v", node.Name, err)
	}

	return concept, nil
}

func createRelationships(ctx context.Context, node ConceptNode) error {
	sourceGUID := generateGUID(ctx, node.Name)
	for _, rel := range node.Relationships {
		targetGUID := generateGUID(ctx, rel.Target)
		relTypeGUID := generateGUID(ctx, rel.Type)
		c := relTypeGUID.AsConcept()
		if c == nil {
			log.Fatalf("Type Description missing: %s in %s\n", rel.Type, node.Name)
		}
		err := createCoreRelationship(ctx, sourceGUID, relTypeGUID, targetGUID)
		if err != nil {
			return fmt.Errorf("failed to create relationship %s -> %s -> %s: %v", node.Name, rel.Type, rel.Target, err)
		}
	}

	for _, child := range node.Children {
		err := createRelationships(ctx, child)
		if err != nil {
			return err
		}
	}

	return nil
}

func BootstrapFromStructure(ctx context.Context, filename string) error {
	structure, err := parseConceptStructure(filename)
	if err != nil {
		return fmt.Errorf("failed to parse concept structure: %v", err)
	}

	// Create relationship types
	for _, rel := range structure.Relationships {
		relationship := &Concept{
			ID:          generateGUID(ctx, rel.Name),
			Name:        rel.Name,
			Description: rel.Description,
			Type:        "RelationshipType",
			Timestamp:   time.Now(),
		}
		err := addOrUpdateConcept(ctx, relationship, peerID)
		if err != nil {
			return fmt.Errorf("failed to add relationship type %s: %v", rel.Name, err)
		}
	}

	// First pass: create all concepts
	for _, node := range structure.Concepts {
		_, err := createConcepts(ctx, node, "")
		if err != nil {
			return fmt.Errorf("failed to create concept %s: %v", node.Name, err)
		}
	}

	// Second pass: create relationships
	for _, node := range structure.Concepts {
		err := createRelationships(ctx, node)
		if err != nil {
			return fmt.Errorf("failed to create relationships for concept %s: %v", node.Name, err)
		}
	}

	log.Printf("Bootstrapped %d concepts and %d relationship types", len(guidMap)-len(structure.Relationships), len(structure.Relationships))
	return nil
}

func createCoreRelationship(ctx context.Context, sourceGUID, relationshipTypeGUID, targetGUID ConceptGUID) error {
	relationshipID := generateGUID(ctx, fmt.Sprintf("%s-%s-%s", sourceGUID, relationshipTypeGUID, targetGUID))

	relationship := &Relationship{
		ID:              relationshipID,
		SourceID:        sourceGUID,
		TargetID:        targetGUID,
		Type:            relationshipTypeGUID,
		EnergyFlow:      1.0,
		FrequencySpec:   []float64{1.0},
		Amplitude:       1.0,
		Volume:          1.0,
		Depth:           1,
		Interactions:    0,
		LastInteraction: time.Now(),
		Timestamp:       time.Now(),
	}

	relationshipMap[relationshipID] = relationship

	// Update the relationships for the source and target concepts
	sourceConcept, exists := conceptMap[sourceGUID]
	if !exists {
		return fmt.Errorf("source concept with GUID %s => (%s) not found", sourceGUID, sourceGUID.findName())
	}
	sourceConcept.Relationships = append(sourceConcept.Relationships, relationshipID)

	targetConcept, exists := conceptMap[targetGUID]
	if !exists {
		return fmt.Errorf("target concept with GUID %s => (%s) not found", targetGUID, targetGUID.findName())
	}
	targetConcept.Relationships = append(targetConcept.Relationships, relationshipID)

	return nil
}

func InitializeSystem(ctx context.Context) error {
	log.Println("Bootstrapping concepts and relationships...")

	if err := BootstrapFromStructure(ctx, "data/concepts_structure.yaml"); err != nil {
		log.Printf("Error during bootstrapping concepts: %v\n", err)
		return err
	}

	log.Println("Concepts and relationships bootstrapped successfully")
	return nil
}
