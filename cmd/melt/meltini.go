package melt

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
)

var meltMelitiniCmd = &cobra.Command{
	Use:              "meltini",
	Short:            "uses meltini APIs to generate metrics from a template",
	Long:             `Uses meltini APIs to generate OT data from provided stated template file.`,
	TraverseChildren: true,
	Run:              meltMeltini,
}

func init() {
	meltMelitiniCmd.Flags().String("template-file", "", "path to the stated template file")
	meltMelitiniCmd.Flags().String("meltini-url", "http://localhost:3000/", "meltini REST APIs URL")
	meltCmd.AddCommand(meltMelitiniCmd) // Assuming meltCmd is defined elsewhere
}

func meltMeltini(cmd *cobra.Command, args []string) {
	templateFile, _ := cmd.Flags().GetString("template-file")
	meltiniURL, _ := cmd.Flags().GetString("meltini-url")

	// Check if template file is provided
	if templateFile == "" {
		fmt.Println("Please provide a path to the stated template file")
		return
	}

	// Read the JSON file
	jsonData, err := os.ReadFile(templateFile)
	if err != nil {
		fmt.Println("Error reading template file:", err)
		return
	}

	// Send a POST request with the JSON data
	response, err := http.Post(meltiniURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error sending request to meltini:", err)
		return
	}
	defer response.Body.Close()

	// Read the response
	respData, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response from meltini:", err)
		return
	}

	// Print the response
	fmt.Println("Response from meltini:", string(respData))
}
