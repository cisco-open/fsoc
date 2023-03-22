package melt

import (
	"io"
	"os"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/melt"
)

// meltCmd represents the login command
var meltSendCmd = &cobra.Command{
	Use:   "push",
	Short: "Generates OTLP telemetry based on fsoc telemetry data model .yaml",
	Long: `This command generates OTLP payload based on a fsoc telemetry data models and sends the data to the FSO Platform Ingestion services.
	
	To properly use the command you will need to create a fsoc profile using an agent principal yaml:
	fsoc config set --profile <agent-principal-profile> --auth agent-principal --secret-file <agent-principal.yaml>
	
	Then you will use the agent principal profile as part of the command:
	fsoc melt push <fsocdatamodel>.yaml --profile <agent-principal-profile> `,
	TraverseChildren: true,
	Run:              meltSend,
}

func getMeltPushCmd() *cobra.Command {
	return meltSendCmd
}

func meltSend(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		log.Error("Missing the fsoc telemetry data model .yaml file name")
		return
	}
	dataFileName := args[0]
	sendDataFromFile(cmd, dataFileName)
}

// func sendDataFromFile(cmd *cobra.Command, dataFileName string) {
// 	fsoData, err := loadDataFile(dataFileName)
// 	if err != nil {
// 		log.Fatalf("Can't open the file named %q: %v", dataFileName, err)
// 	}
// 	cfg := config.GetCurrentContext()
// 	authObj, err := api.AgentPrincipalLogin(cfg, fsoData.Credentials)
// 	if err != nil {
// 		log.Fatalf("Couldn't authenticate using agent principal credentials: %v", err)
// 	}

// 	// authObj := &api.TokenStruct{
// 	// 	AccessToken: "thisIsTheToken",
// 	// }
// 	exportMeltStraight(cmd, authObj, fsoData)
// }

func sendDataFromFile(cmd *cobra.Command, dataFileName string) {
	fsoData, err := loadDataFile(dataFileName)
	if err != nil {
		log.Fatalf("Can't open the file named %q: %v", dataFileName, err)
	}
	exportMeltStraight(cmd, fsoData)
}

// func exportMeltStraight(cmd *cobra.Command, authObj *api.TokenStruct, fsoData *melt.FsocData) {

// 	token := authObj.AccessToken
// 	endPoint := "/data/v1"

// 	output.PrintCmdStatus(cmd, "Generating new MELT telemetry... \n")
// 	exportMelt(cmd, endPoint, token, *fsoData)
// }

func exportMeltStraight(cmd *cobra.Command, fsoData *melt.FsocData) {
	output.PrintCmdStatus(cmd, "Generating new MELT telemetry... \n")
	exportMelt(cmd, *fsoData)
}

// func exportMeltTicker(cmd *cobra.Command, fsoData *melt.FsoData) {
// 	interval, _ := time.ParseDuration("1m")
// 	ticker := time.NewTicker(interval)

// 	for _ = range ticker.C {

// 		authObj, err := login.Login(fsoData.Credentials)
// 		if err != nil {
// 			log.Fatalf("Failed to authenticate %v", err.Error())
// 			return
// 		}

// 		token := authObj.AccessToken
// 		urlStruct, err := url.Parse(fsoData.Credentials.TokenURL)
// 		server := urlStruct.Host

// 		endPoint := "https://" + server + "/data/v1"

// 		output.PrintCmdStatus("Generating and pushing OTLP MELT Telemetry data to the FSO Platform...")
// 		exportMelt(cmd, endPoint, token, *fsoData)
// 	}
// }

// func exportMelt(cmd *cobra.Command, endPoint, token string, fsoData melt.FsocData) {
// 	// invoke the exporter
// 	exp := &melt.Exporter{}

// 	output.PrintCmdStatus(cmd, "\nExporting metrics... \n")
// 	err := exp.ExportMetrics(fsoData.Melt)
// 	if err != nil {
// 		log.Fatalf("Error exporting metrics: %s", err)
// 	}

// 	output.PrintCmdStatus(cmd, "\nExporting logs... \n")
// 	err = exp.ExportLogs(fsoData.Melt)
// 	if err != nil {
// 		log.Fatalf("Error exporting logs: %s", err)
// 	}

// }

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

// func exportEvents(ctx context.Context, endPoint, token string) error {
// 	el := []*melt.Entity{}

// 	// create an entity
// 	e1 := melt.NewEntity("geometry:square")
// 	e1.SetAttribute("geometry.shape.name", "Square 100")
// 	e1.SetAttribute("geometry.shape.type", "square")
// 	e1.SetAttribute("geometry.square.side", "10")
// 	e1.SetAttribute("telemetry.sdk.name", "appd-datagen")

// 	// add events to the entity
// 	for i := 1; i < 5; i++ {
// 		l := melt.NewEvent("geometry:operation")
// 		l.SetAttribute("type", "draw")
// 		l.Timestamp = time.Now().UnixNano()
// 		e1.AddLog(l)
// 	}
// 	el = append(el, e1)

// 	// invoke the exporter
// 	exp := &exporter.Exporter{}
// 	expReq := melt.ExportRequest{
// 		EndPoint:    endPoint,
// 		Credentials: melt.Credentials{Token: token},
// 	}
// 	return exp.ExportLogs(ctx, expReq, el)
// }
