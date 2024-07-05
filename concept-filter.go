package main

import (
	"strings"
)

func isEmptyFilter(filter ConceptFilter) bool {
	return filter.CID == "" && filter.GUID == "" && filter.Name == "" &&
		filter.Description == "" && filter.Type == "" && filter.TimestampAfter == nil
}

func matchesConcept(concept Concept, filter ConceptFilter) bool {
	if filter.CID != "" && string(concept.GetCID()) != filter.CID {
		return false
	}
	if filter.GUID != "" && concept.GetGUID() != filter.GUID {
		return false
	}
	if filter.Name != "" && !strings.Contains(strings.ToLower(concept.GetName()), strings.ToLower(filter.Name)) {
		return false
	}
	if filter.Description != "" && !strings.Contains(strings.ToLower(concept.GetDescription()), strings.ToLower(filter.Description)) {
		return false
	}
	if filter.Type != "" && concept.GetType() != filter.Type {
		return false
	}
	if filter.TimestampAfter != nil && !concept.GetTimestamp().After(*filter.TimestampAfter) {
		return false
	}
	return true
}

func filterConcepts(filter ConceptFilter) []Concept {
	conceptMu.RLock()
	defer conceptMu.RUnlock()

	if isEmptyFilter(filter) {
		concepts := make([]Concept, 0, len(conceptMap))
		for _, concept := range conceptMap {
			concepts = append(concepts, *concept)
		}
		return concepts
	}

	var filteredConcepts []Concept
	for _, concept := range conceptMap {
		if matchesConcept(*concept, filter) {
			filteredConcepts = append(filteredConcepts, *concept)
		}
	}
	return filteredConcepts
}
