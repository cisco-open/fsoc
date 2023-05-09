// Copyright 2023 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package solution

import (
	"fmt"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

func getResourceMap(cmd *cobra.Command, entityName string, manifest *Manifest) *FmmResourceMapping {
	var newResoureMapping *FmmResourceMapping

	entity := findEntity(entityName, manifest)
	if entity == nil {
		log.Fatalf("Couldn't find an entity type named %s", entityName)
	}

	namespace := entity.Namespace
	name := fmt.Sprintf("%s_%s_entity_mapping", manifest.Name, entityName)
	entityType := fmt.Sprintf("%s:%s", manifest.Name, entityName)
	scopeFilterFields := make([]string, 0)
	attributeMaps := make(FmmNameMappings, 0)
	displayName := fmt.Sprintf("Resource mapping configuration for the %q entity", entityType)
	fmmTypeDef := &FmmTypeDef{
		Namespace:   namespace,
		Kind:        "resourceMapping",
		Name:        name,
		DisplayName: displayName,
	}

	for _, requiredField := range entity.AttributeDefinitions.Required {
		scopeForField := fmt.Sprintf("%s.%s.%s", manifest.Name, entityName, requiredField)
		scopeFilterFields = append(scopeFilterFields, scopeForField)
	}

	for k, _ := range entity.AttributeDefinitions.Attributes {
		scopeForField := fmt.Sprintf("%s.%s.%s", manifest.Name, entityName, k)
		attributeMaps[k] = scopeForField
	}

	scopeFilter := fmt.Sprintf("containsAll(resourceAttributes, %s)", getStringfiedArray(scopeFilterFields))
	newResoureMapping = &FmmResourceMapping{
		FmmTypeDef:            fmmTypeDef,
		EntityType:            entityType,
		ScopeFilter:           scopeFilter,
		AttributeNameMappings: attributeMaps,
	}

	return newResoureMapping
}

func getAssociationDeclarations(entityName string, manifest *Manifest) []*FmmAssociationDeclaration {
	entity := findEntity(entityName, manifest)
	if entity == nil {
		log.Fatalf("Couldn't find an entity type named %s", entityName)
	}
	fmmAssocDeclarations := make([]*FmmAssociationDeclaration, 0)

	if entity.AssociationTypes != nil {
		for _, toType := range entity.AssociationTypes.Consists_of {
			fmmAssocDeclarations = append(fmmAssocDeclarations, getAssociationDeclaration(entity, "common:consists_of", toType))
		}
		for _, toType := range entity.AssociationTypes.Aggregates_of {
			fmmAssocDeclarations = append(fmmAssocDeclarations, getAssociationDeclaration(entity, "common:aggregates_of", toType))
		}
		for _, toType := range entity.AssociationTypes.Is_a {
			fmmAssocDeclarations = append(fmmAssocDeclarations, getAssociationDeclaration(entity, "common:is_a", toType))
		}
		for _, toType := range entity.AssociationTypes.Has {
			fmmAssocDeclarations = append(fmmAssocDeclarations, getAssociationDeclaration(entity, "common:has", toType))
		}
		for _, toType := range entity.AssociationTypes.Relates_to {
			fmmAssocDeclarations = append(fmmAssocDeclarations, getAssociationDeclaration(entity, "common:relates_to", toType))
		}
		for _, toType := range entity.AssociationTypes.Uses {
			fmmAssocDeclarations = append(fmmAssocDeclarations, getAssociationDeclaration(entity, "common:uses", toType))
		}
	}

	return fmmAssocDeclarations
}

func getAssociationDeclaration(entity *FmmEntity, associationType string, toType string) *FmmAssociationDeclaration {

	toTypeSplit := strings.Split(toType, ":")
	toTypeDesc := toType
	if toTypeSplit[0] == entity.Namespace.Name {
		toTypeDesc = toTypeSplit[1]
	} else {
		toTypeDesc = fmt.Sprintf("%s_%s", toTypeSplit[0], toTypeSplit[1])
	}

	fmmTypeDef := &FmmTypeDef{
		Namespace:   entity.Namespace,
		Kind:        "associationDeclaration",
		Name:        fmt.Sprintf("%s_to_%s_relationship", entity.Name, toTypeDesc),
		DisplayName: fmt.Sprintf("Declared Relationship between %s and %s", entity.Name, strings.Replace(toTypeDesc, "_", ":", 1)),
	}
	declaration := &FmmAssociationDeclaration{
		FmmTypeDef:      fmmTypeDef,
		ScopeFilter:     "true",
		FromType:        entity.GetTypeName(),
		ToType:          toType,
		AssociationType: associationType,
	}
	return declaration
}

func findEntity(entityName string, manifest *Manifest) *FmmEntity {
	entities := manifest.GetFmmEntities()
	var entity *FmmEntity
	for _, e := range entities {
		if e.Name == entityName {
			entity = e
			break
		}
	}
	if entity == nil {
		log.Fatalf("Couldn't find an entity type named %s", entityName)
	}
	return entity
}

func getNamespaceComponent(solutionName string) *FmmNamespace {
	namespaceDef := &FmmNamespace{
		Name: solutionName,
	}
	return namespaceDef
}

func getEntityComponent(entityName string, namespaceName string) *FmmEntity {
	emptyStringArray := make([]string, 0)
	emptyAttributeArray := make(map[string]*FmmAttributeTypeDef, 1)
	// emptyAssociationTypes := &FmmAssociationTypesTypeDef{}

	emptyAttributeArray["name"] = &FmmAttributeTypeDef{
		Type:        "string",
		Description: fmt.Sprintf("The name of the %s", entityName),
	}

	namespaceAssign := &FmmNamespaceAssignTypeDef{
		Name:    namespaceName,
		Version: 1,
	}

	lifecycleConfig := &FmmLifecycleConfigTypeDef{
		PurgeTtlInMinutes:     4200,
		RetentionTtlInMinutes: 1440,
	}

	fmmTypeDef := &FmmTypeDef{
		Namespace:   *namespaceAssign,
		Kind:        "entity",
		Name:        entityName,
		DisplayName: entityName,
	}

	requiredArray := append(emptyStringArray, "name")
	attributesDefinition := &FmmAttributeDefinitionsTypeDef{
		Required:   requiredArray,
		Optimized:  emptyStringArray,
		Attributes: emptyAttributeArray,
	}

	entityComponentDef := &FmmEntity{
		FmmTypeDef:            fmmTypeDef,
		LifecyleConfiguration: lifecycleConfig,
		AttributeDefinitions:  attributesDefinition,
	}

	return entityComponentDef
}

func getEventComponent(eventName string, namespaceName string) *FmmEvent {
	emptyStringArray := make([]string, 0)
	emptyAttributeArray := make(map[string]*FmmAttributeTypeDef, 1)

	emptyAttributeArray["name"] = &FmmAttributeTypeDef{
		Type:        "string",
		Description: fmt.Sprintf("The name of the %s", eventName),
	}

	namespaceAssign := &FmmNamespaceAssignTypeDef{
		Name:    namespaceName,
		Version: 1,
	}

	fmmTypeDef := &FmmTypeDef{
		Namespace:   *namespaceAssign,
		Kind:        "event",
		Name:        eventName,
		DisplayName: eventName,
	}

	requiredArray := append(emptyStringArray, "name")
	attributesDefinition := &FmmAttributeDefinitionsTypeDef{
		Required:   requiredArray,
		Optimized:  emptyStringArray,
		Attributes: emptyAttributeArray,
	}

	eventComponentDef := &FmmEvent{
		FmmTypeDef:           fmmTypeDef,
		AttributeDefinitions: attributesDefinition,
	}

	return eventComponentDef
}

func getMetricComponent(metricName string, contentType FmmMetricContentType, category FmmMetricCategory, metricType FmmMetricType, namespaceName string) *FmmMetric {
	namespaceAssign := &FmmNamespaceAssignTypeDef{
		Name:    namespaceName,
		Version: 1,
	}

	fmmTypeDef := &FmmTypeDef{
		Namespace:   *namespaceAssign,
		Kind:        "metric",
		Name:        metricName,
		DisplayName: metricName,
	}

	metricComponentDef := &FmmMetric{
		FmmTypeDef:             fmmTypeDef,
		Category:               category,
		ContentType:            contentType,
		AggregationTemporality: "delta",
		IsMonotonic:            false,
		Type:                   metricType,
		Unit:                   "{Count}",
	}

	return metricComponentDef
}

func getServiceComponent(serviceName string) *ServiceDef {
	serviceComponentDef := &ServiceDef{
		Name:  serviceName,
		Image: "dockerRegistryURL",
	}

	return serviceComponentDef
}

func checkCreateSolutionNamespace(cmd *cobra.Command, manifest *Manifest, folderName string) {
	componentType := "fmm:namespace"
	namespaceName := manifest.Name
	fileName := namespaceName + ".json"
	objFilePath := fmt.Sprintf("%s/%s", folderName, fileName)

	componentDef := manifest.GetComponentDef(componentType)

	if componentDef.Type == "" {
		addCompDefToManifest(cmd, manifest, componentType, folderName)

		if _, err := os.Stat(objFilePath); os.IsNotExist(err) {
			namespaceComp := getNamespaceComponent(namespaceName)
			createComponentFile(namespaceComp, folderName, fileName)
			statusMsg := fmt.Sprintf("Added %s file to your solution \n", objFilePath)
			output.PrintCmdStatus(cmd, statusMsg)
		}

	}

}
