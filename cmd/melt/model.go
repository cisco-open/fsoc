package melt

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v2"

	sol "github.com/cisco-open/fsoc/cmd/solution"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/melt"
)

// meltCmd represents the login command
var meltModelCmd = &cobra.Command{
	Use:              "model",
	Short:            "Generates a fsoc telemetry data model .yaml file based on your solution domain model",
	Long:             `This command converts your fmm domain models as a fsoc telemetry data model .yaml file so you can generate mock telemetry data for your solutioin.`,
	TraverseChildren: true,
	Run:              meltModel,
}

func init() {
	meltCmd.AddCommand(meltModelCmd)
}

func meltModel(cmd *cobra.Command, args []string) {
	manifest := sol.GetManifest()
	fileName := fmt.Sprintf("%s-%s-melt.yaml", manifest.Name, manifest.SolutionVersion)
	fsocData := getFsocDataModel(cmd, manifest)
	output.PrintCmdStatus(cmd, fmt.Sprintf("Generating %s\n", fileName))
	writeDataFile(fsocData, fileName)
}

func getFsocDataModel(cmd *cobra.Command, manifest *sol.Manifest) *melt.FsocData {
	fsocData := &melt.FsocData{}
	fmmEntities := GetFmmEntities(manifest)
	output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %v entities to the fsoc data model\n", len(fmmEntities)))
	fmmMetrics := GetFmmMetrics(manifest)
	output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %v metrics to the fsoc data model\n", len(fmmMetrics)))
	fsocMetrics := GetFsocMetrics(fmmMetrics)
	fsocEntities := GetFsocEntities(fmmEntities, fsocMetrics)
	fsocData.Melt = fsocEntities

	return fsocData
}

func GetFmmMetrics(manifest *sol.Manifest) []*sol.FmmMetric {
	fmmMetrics := make([]*sol.FmmMetric, 0)
	entityComponentDefs := GetComponentDefs("fmm:metric", manifest)
	for _, compDef := range entityComponentDefs {
		if compDef.ObjectsFile != "" {
			filePath := compDef.ObjectsFile
			fmmMetrics = append(fmmMetrics, getFmmMetricsFromFile(filePath)...)
		}
		if compDef.ObjectsDir != "" {
			filePath := compDef.ObjectsDir
			err := filepath.Walk(filePath,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if strings.Contains(path, ".json") {
						fmmMetrics = append(fmmMetrics, getFmmMetricsFromFile(path)...)
					}
					return nil
				})
			if err != nil {
				log.Fatalf("Error traversing the folder: %v", err)
			}
		}

	}
	return fmmMetrics
}

func getFmmMetricsFromFile(filePath string) []*sol.FmmMetric {
	fmmMetrics := make([]*sol.FmmMetric, 0)
	metricDefFile := openFile(filePath)
	defer metricDefFile.Close()
	metricDefBytes, _ := io.ReadAll(metricDefFile)
	metricDefContent := string(metricDefBytes)

	if strings.Index(metricDefContent, "[") == 0 {
		metricsArray := make([]*sol.FmmMetric, 0)
		err := json.Unmarshal(metricDefBytes, &metricsArray)
		if err != nil {
			log.Fatalf("Can't parse an array of metric definition objects from the %q file:\n %v", filePath, err)
		}
		fmmMetrics = append(fmmMetrics, metricsArray...)
	} else {
		var metric *sol.FmmMetric
		err := json.Unmarshal(metricDefBytes, &metric)
		if err != nil {
			log.Fatalf("Can't parse a metric definition objects from the %q file:\n %v ", filePath, err)
		}
		fmmMetrics = append(fmmMetrics, metric)
	}
	return fmmMetrics
}

func GetFsocMetrics(fmmMetrics []*sol.FmmMetric) []*melt.Metric {
	fsocMetrics := make([]*melt.Metric, 0)

	for _, fmmMt := range fmmMetrics {
		fsocMType := fmt.Sprintf("%s:%s", fmmMt.Namespace.Name, fmmMt.Name)
		fsocMt := melt.NewMetric(fsocMType, fmmMt.Unit, string(fmmMt.ContentType), string(fmmMt.Type))
		fsocMetrics = append(fsocMetrics, fsocMt)
	}

	return fsocMetrics
}

func GetFsocEntities(fmmEntities []*sol.FmmEntity, fsocMetrics []*melt.Metric) []*melt.Entity {
	fsocEntities := make([]*melt.Entity, 0)

	for _, fmmE := range fmmEntities {
		fsocEType := fmt.Sprintf("%s:%s", fmmE.Namespace.Name, fmmE.Name)
		fsocE := melt.NewEntity(fsocEType)
		fmmAttrs := maps.Keys(fmmE.AttributeDefinitions.Attributes)
		for _, fmmAttr := range fmmAttrs {
			if !strings.Contains(fmmAttr, fmmE.Namespace.Name) {
				fmmAttr = fmt.Sprintf("%s.%s.%s", fmmE.Namespace.Name, fmmE.Name, fmmAttr)
			}
			fsocE.SetAttribute(fmmAttr, "")
		}

		//adding fsoc metrics to the model
		for _, fmmEM := range fmmE.MetricTypes {
			fsocMetricType := fmmEM
			for _, fsocMt := range fsocMetrics {
				if fsocMt.TypeName == fsocMetricType {
					fsocE.AddMetric(fsocMt)
				}
			}

		}

		// add logs tp the entity
		for i := 0; i < 2; i++ {
			fsocL := melt.NewLog()
			fsocL.SetAttribute("level", "info")
			fsocL.Severity = "INFO"
			fsocL.Body = fmt.Sprintf("hello world-%d for an entity of type %s", i, fsocE.TypeName)
			fsocE.AddLog(fsocL)
		}

		fsocEntities = append(fsocEntities, fsocE)
	}

	return fsocEntities
}

func GetFmmEntities(manifest *sol.Manifest) []*sol.FmmEntity {
	fmmEntities := make([]*sol.FmmEntity, 0)
	entityComponentDefs := GetComponentDefs("fmm:entity", manifest)
	for _, compDef := range entityComponentDefs {
		if compDef.ObjectsFile != "" {
			filePath := compDef.ObjectsFile
			fmmEntities = append(fmmEntities, getFmmEntitiesFromFile(filePath)...)
		}
		if compDef.ObjectsDir != "" {
			filePath := compDef.ObjectsDir
			err := filepath.Walk(filePath,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if strings.Contains(path, ".json") {
						fmmEntities = append(fmmEntities, getFmmEntitiesFromFile(path)...)
					}
					return nil
				})
			if err != nil {
				log.Fatalf("Error traversing the folder: %v", err)
			}
		}

	}
	return fmmEntities
}

func getFmmEntitiesFromFile(filePath string) []*sol.FmmEntity {
	fmmEntities := make([]*sol.FmmEntity, 0)
	entityDefFile := openFile(filePath)
	defer entityDefFile.Close()
	entityDefBytes, _ := io.ReadAll(entityDefFile)
	entityDefContent := string(entityDefBytes)

	if strings.Index(entityDefContent, "[") == 0 {
		entitiesArray := make([]*sol.FmmEntity, 0)
		err := json.Unmarshal(entityDefBytes, &entitiesArray)
		if err != nil {
			log.Fatalf("Can't parse an array of entity definition objects from the %q file:\n %v", filePath, err)
		}
		fmmEntities = append(fmmEntities, entitiesArray...)
	} else {
		var entity *sol.FmmEntity
		err := json.Unmarshal(entityDefBytes, &entity)
		if err != nil {
			log.Fatalf("Can't parse an entity definition objects from the %q file:\n %v", filePath, err)
		}
		fmmEntities = append(fmmEntities, entity)
	}
	return fmmEntities
}

func GetComponentDefs(typeName string, manifest *sol.Manifest) []sol.ComponentDef {

	componentDefs := make([]sol.ComponentDef, 0)

	for _, compDef := range manifest.Objects {
		if compDef.Type == typeName {
			componentDefs = append(componentDefs, compDef)
		}
	}
	return componentDefs
}

func openFile(filePath string) *os.File {
	svcFile, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Can't open the file named %q: %v", filePath, err)
	}
	return svcFile
}

func writeDataFile(fsoData *melt.FsocData, fileName string) {
	fsoDataYamlFile, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Failed to create FsoData yaml file %q: %v", fileName, err)
	}
	defer fsoDataYamlFile.Close()

	svcJson, _ := yaml.Marshal(fsoData)

	_, _ = fsoDataYamlFile.WriteString(string(svcJson))
	fsoDataYamlFile.Close()
}
