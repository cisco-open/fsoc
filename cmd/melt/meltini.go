package melt

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var meltiniCmd = &cobra.Command{
	Use:   "meltini",
	Short: "Run a JSONata expression from a file",
	Run:   runMeltini,
}

func init() {
	meltCmd.AddCommand(meltiniCmd)
	meltiniCmd.Flags().StringP("file", "f", "", "Input file with JSONata expression")
}

func runMeltini(cmd *cobra.Command, args []string) {
	filename, _ := cmd.Flags().GetString("file")

	jsonataExpr, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
	defer os.RemoveAll("temp.json")
	jsonData, err := os.ReadFile("data.json")
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
	// Create a temporary JavaScript file

	// Write JavaScript code to the file
	jsCode := fmt.Sprintf(`
		const jsonata = require('jsonata');
		const data = %s;
		const expr = jsonata(%q);
		async function runAsyncShit() {
		  try {
			const result = await expr.evaluate(data);
			console.log(JSON.stringify(result));
		  } catch (err) {
			console.error(err);
		  }
		}
		runAsyncShit();
	`, string(jsonData), string(jsonataExpr))

	err = ioutil.WriteFile("temp.js", []byte(jsCode), 0644)

	if err != nil {
		fmt.Printf("Error writing to temporary file: %v", err)
		os.Exit(1)
	}

	// Run the temporary JS file with Node.js
	cmdNode := exec.Command("node", "temp.js")
	output, err := cmdNode.CombinedOutput()
	if err != nil {
		log.Fatalf("Error running Node.js script: {output: %v, err: %v}", output, err)
	}

	// Print the result
	fmt.Printf("Result: %s", output)
}
