package melt

import (
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/melt"
)

var meltPushCmd = &cobra.Command{
	Use:   "push DATAFILE",
	Short: "Generates OTLP telemetry based on fsoc telemetry data model .yaml",
	Long: `This command generates OTLP payload based on a fsoc telemetry data models and sends the data to the FSO Platform Ingestion services.
	
	To properly use the command you will need to create a fsoc profile using an agent principal yaml:
	fsoc config set --profile=<agent-principal-profile> auth=agent-principal secret-file=<agent-principal.yaml>
	
	Then you will use the agent principal profile as part of the command:
	fsoc melt push <fsocdatamodel>.yaml --profile <agent-principal-profile> `,
	TraverseChildren: true,
	Args:             cobra.ExactArgs(1),
	Run:              meltSend,
}

func init() {
	meltCmd.AddCommand(meltPushCmd)
}

func meltSend(cmd *cobra.Command, args []string) {
	ctx := config.GetCurrentContext()
	if ctx.AuthMethod != config.AuthMethodAgentPrincipal {
		_ = cmd.Help()
		log.Fatalf("This command requires a profile with \"agent-principal\" auth method, found %q instead", ctx.AuthMethod)
	}
	dataFileName := args[0]
	sendDataFromFile(cmd, dataFileName)
}

func sendDataFromFile(cmd *cobra.Command, dataFileName string) {
	fsoData, err := loadDataFile(dataFileName)
	if err != nil {
		log.Fatalf("Can't open data file %q: %v", dataFileName, err)
	}

	for _, entity := range fsoData.Melt {
		if _, ok := entity.Attributes["telemetry.sdk.name"]; ok {
			log.Info("telemetry.sdk.name already set, skipping...")
		} else {
			entity.SetAttribute("telemetry.sdk.name", "fsoc-melt")
		}
		for _, m := range entity.Metrics {
			et := time.Now()
			if len(m.DataPoints) == 0 {
				for i := 1; i < 6; i++ {
					st := et.Add(time.Minute * -1)
					m.AddDataPoint(st.UnixNano(), et.UnixNano(), rand.Float64()*50)
					et = st
				}
			}
		}
		for _, l := range entity.Logs {
			if l.Timestamp == 0 {
				l.Timestamp = time.Now().UnixNano()
			}
		}
	}

	exportMeltStraight(cmd, fsoData)
}

func exportMeltStraight(cmd *cobra.Command, fsoData *melt.FsocData) {
	output.PrintCmdStatus(cmd, "Generating new MELT telemetry... \n")
	exportMelt(cmd, *fsoData)
}

func exportMelt(cmd *cobra.Command, fsoData melt.FsocData) {
	// invoke the exporter
	exp := &melt.Exporter{}

	output.PrintCmdStatus(cmd, "\nExporting metrics... \n")
	err := exp.ExportMetrics(fsoData.Melt)
	if err != nil {
		log.Fatalf("Error exporting metrics: %s", err)
	}

	output.PrintCmdStatus(cmd, "\nExporting logs... \n")
	err = exp.ExportLogs(fsoData.Melt)
	if err != nil {
		log.Fatalf("Error exporting logs: %s", err)
	}

}

func loadDataFile(fileName string) (*melt.FsocData, error) {
	var fsoData *melt.FsocData

	dataFile, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Can't open the file named %q: %v", fileName, err)
	}

	defer dataFile.Close()

	dataBytes, _ := io.ReadAll(dataFile)

	err = yaml.Unmarshal(dataBytes, &fsoData)
	if err != nil {
		log.Fatalf("Failed to parse fsoc telemetry model file: %v", err)
	}

	return fsoData, nil
}
