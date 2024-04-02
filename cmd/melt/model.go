package melt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/afero"
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
	meltModelCmd.Flags().String("tag", "", "tag for the solution (only for pseudo-isolated solutions, DEPRECATED)")
	meltModelCmd.Flags().String("env-file", "", "path to the env vars json file (only for pseudo-isolated solutions, DEPRECATED)")
	meltModelCmd.MarkFlagsMutuallyExclusive("tag", "env-file")

	meltCmd.AddCommand(meltModelCmd)
}

func meltModel(cmd *cobra.Command, args []string) {
	manifest, err := sol.GetManifest(".")
	if err != nil {
		log.Fatalf("Failed to get manifest: %v", err)
	}
	if manifest.HasPseudoIsolation() {
		// determine tag
		tag, envFile := sol.DetermineTagEnvFile(cmd, ".") // --tag, --stable, --env-file, FSOC_SOLUTION_TAG env var, .tag file
		if tag == "" && envFile == "" {
			log.Fatal("A tag must be specified for modeling in pseudo-isolated solutions")
		}
		if tag != "" {
			if !sol.IsValidSolutionTag(tag) {
				log.Fatalf("Invalid tag %q", tag)
			}
		}
		envVars, err := sol.LoadEnvVars(cmd, tag, envFile) // the tag now remains in the envVars (whether read from file or synthesized)
		if err != nil {
			log.Fatalf("Failed to define isolation environment: %v", err)
		}

		currentDirectory, err := filepath.Abs(".")
		if err != nil {
			log.Fatalf("Error getting current directory: %v", err)
		}
		fileSystemRoot := afero.NewBasePathFs(afero.NewOsFs(), currentDirectory)

		isolateNamespace := fmt.Sprintf("%s%s", strings.Split(manifest.Name, "$")[0], sol.GetPseudoIsolationTag(envVars))
		fileName := fmt.Sprintf("%s-%s-melt.yaml", isolateNamespace, manifest.SolutionVersion)

		fsocData := getFsocDataModel(cmd, manifest, isolateNamespace)
		output.PrintCmdStatus(cmd, fmt.Sprintf("Generating %s\n", fileName))
		writeDataFile(fsocData, fileName)

		err = sol.ReplaceStringInFile(fileSystemRoot, fileName, "${sys.solutionId}", isolateNamespace)
		if err != nil {
			log.Fatalf("Error isolating melt model file: %v", err)
		}
	} else {
		fileName := fmt.Sprintf("%s-%s-melt.yaml", manifest.Name, manifest.SolutionVersion)
		fsocData := getFsocDataModel(cmd, manifest, "")
		output.PrintCmdStatus(cmd, fmt.Sprintf("Generating %s\n", fileName))
		writeDataFile(fsocData, fileName)
	}

}

func getFsocDataModel(cmd *cobra.Command, manifest *sol.Manifest, isolationNamespace string) *melt.FsocData {
	fsocData := &melt.FsocData{}
	fmmEntities := manifest.GetFmmEntities()
	output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %v entities to the fsoc data model\n", len(fmmEntities)))

	fmmMetrics := manifest.GetFmmMetrics()
	output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %v metrics to the fsoc data model\n", len(fmmMetrics)))

	fmmEvents := manifest.GetFmmEvents()
	output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %v events to the fsoc data model\n", len(fmmEvents)))

	if isolationNamespace != "" {
		realNamespace := manifest.GetSolutionName()

		for _, e := range fmmEntities {
			e.Namespace.Name = isolationNamespace

			fmmAttrsDefs := e.AttributeDefinitions.Attributes
			for k, v := range fmmAttrsDefs {
				if strings.Contains(k, realNamespace) {
					entityAttr := k[len(realNamespace)+1:]
					newKey := fmt.Sprintf("%s.%s", e.Namespace.Name, entityAttr)
					newValue := v
					delete(fmmAttrsDefs, k)
					fmmAttrsDefs[newKey] = newValue
				}
			}

			metricRefs := e.MetricTypes
			e.MetricTypes = GetIsolatedRefs(metricRefs, manifest, isolationNamespace)

			evtRefs := e.EventTypes
			e.EventTypes = GetIsolatedRefs(evtRefs, manifest, isolationNamespace)

		}
		for _, m := range fmmMetrics {
			m.Namespace.Name = isolationNamespace
		}
		for _, evt := range fmmEvents {
			evt.Namespace.Name = isolationNamespace
		}
	}

	fsocMetrics := GetFsocMetrics(fmmMetrics)
	fsocEvents := GetFsocEvents(fmmEvents)
	fsocEntities := GetFsocEntities(fmmEntities, fsocMetrics, fsocEvents)
	fsocData.Melt = fsocEntities

	return fsocData
}

func GetIsolatedRefs(fmmTypeRefs []string, manifest *sol.Manifest, isolationNamespace string) []string {
	newFmmTypeRefs := make([]string, 0)
	for _, typeRef := range fmmTypeRefs {
		fmmTypeConvention := strings.Split(typeRef, ":")
		if strings.Contains(fmmTypeConvention[0], manifest.GetNamespaceName()) {
			isolateMetricRef := fmt.Sprintf("%s:%s", isolationNamespace, fmmTypeConvention[1])
			newFmmTypeRefs = append(newFmmTypeRefs, isolateMetricRef)
		}
	}
	return newFmmTypeRefs
}

func GetFsocEvents(fmmEvents []*sol.FmmEvent) []*melt.Log {
	fsocEvents := make([]*melt.Log, 0)

	for _, fmmEvt := range fmmEvents {
		fsocEType := fmt.Sprintf("%s:%s", fmmEvt.Namespace.Name, fmmEvt.Name)
		fsocEvt := melt.NewEvent(fsocEType)
		for attrName, attrTypeDef := range fmmEvt.AttributeDefinitions.Attributes {
			fsocEvt.SetAttribute(attrName, getDefaultValue(attrTypeDef))
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
			for attrName, attrTypeDef := range fmmMt.AttributeDefinitions.Attributes {
				fsocMt.SetAttribute(attrName, getDefaultValue(attrTypeDef))
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
			origName := fmmAttr
			if !strings.Contains(fmmAttr, fmmE.Namespace.Name) {
				fmmAttr = fmt.Sprintf("%s.%s.%s", fmmE.Namespace.Name, fmmE.Name, fmmAttr)
			}
			fsocE.SetAttribute(fmmAttr, getDefaultValue(fmmE.AttributeDefinitions.Attributes[origName]))
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

func getDefaultValue(td *sol.FmmAttributeTypeDef) interface{} {
	switch td.Type {
	case "boolean":
		return false
	case "long":
		return 1
	case "double":
		return 1.1
	default:
		return ""
	}
}
