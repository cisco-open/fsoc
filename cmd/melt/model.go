package melt

import (
	"fmt"
	"os"
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
	fmmEntities := manifest.GetFmmEntities()
	output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %v entities to the fsoc data model\n", len(fmmEntities)))

	fmmMetrics := manifest.GetFmmMetrics()
	output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %v metrics to the fsoc data model\n", len(fmmMetrics)))
	fsocMetrics := GetFsocMetrics(fmmMetrics)

	fmmEvents := manifest.GetFmmEvents()
	output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %v events to the fsoc data model\n", len(fmmEvents)))
	fsocEvents := GetFsocEvents(fmmEvents)

	fsocEntities := GetFsocEntities(fmmEntities, fsocMetrics, fsocEvents)
	fsocData.Melt = fsocEntities

	return fsocData
}

func GetFsocEvents(fmmEvents []*sol.FmmEvent) []*melt.Log {
	fsocEvents := make([]*melt.Log, 0)

	for _, fmmEvt := range fmmEvents {
		fsocEType := fmt.Sprintf("%s:%s", fmmEvt.Namespace.Name, fmmEvt.Name)
		fsocEvt := melt.NewEvent(fsocEType)
		fmmAttrs := maps.Keys(fmmEvt.AttributeDefinitions.Attributes)

		for _, fmmAttr := range fmmAttrs {
			fsocEvt.SetAttribute(fmmAttr, "")
		}

		fsocEvents = append(fsocEvents, fsocEvt)
	}
	return fsocEvents
}

func GetFsocMetrics(fmmMetrics []*sol.FmmMetric) []*melt.Metric {
	fsocMetrics := make([]*melt.Metric, 0)

	for _, fmmMt := range fmmMetrics {
		fsocMType := fmt.Sprintf("%s:%s", fmmMt.Namespace.Name, fmmMt.Name)
		fsocMt := melt.NewMetric(fsocMType, fmmMt.Unit, string(fmmMt.ContentType), string(fmmMt.Type))
		if fmmMt.AttributeDefinitions != nil {
			fmmAttrs := maps.Keys(fmmMt.AttributeDefinitions.Attributes)
			for _, fmmAttr := range fmmAttrs {
				fsocMt.SetAttribute(fmmAttr, "")
			}
		}
		fsocMetrics = append(fsocMetrics, fsocMt)
	}

	return fsocMetrics
}

func GetFsocEntities(fmmEntities []*sol.FmmEntity, fsocMetrics []*melt.Metric, fsocEvents []*melt.Log) []*melt.Entity {
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

		//adding fsoc events to the model
		for _, fmmEM := range fmmE.EventTypes {
			fsocEventType := fmmEM
			for _, fsocEvt := range fsocEvents {
				if fsocEvt.TypeName == fsocEventType {
					fsocE.AddLog(fsocEvt)
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
