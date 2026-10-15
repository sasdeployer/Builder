package utils

import (
	"Builder/spinner"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

func PushBuildData() {
	pushURL := os.Getenv("BUILDER_PUSH_URL")
	artifactDir := os.Getenv("BUILDER_ARTIFACT_DIR")

	// Read in build JSON data from build artifact directory
	metadataJSON, errMetadataRead := os.ReadFile(artifactDir + "/metadata.json")
	if errMetadataRead != nil {
		spinner.LogMessage("Can't find / open metadata file: "+errMetadataRead.Error(), "fatal")
	}

	// Read in logs JSON data from logs directory and add to metadata json
	lastSlash := strings.LastIndex(artifactDir, "/")
	logsFilePath := artifactDir[0:lastSlash] + "/logs/logs.json"
	validLogsJSONString := getLogsJSON(logsFilePath)

	var bodyFormat map[string]interface{}
	json.Unmarshal(metadataJSON, &bodyFormat)

	bodyFormat["logs"] = validLogsJSONString

	bodyJSON, errBodyMarshal := json.Marshal(bodyFormat)
	if errBodyMarshal != nil {
		spinner.LogMessage("Error encoding JSON body: "+errBodyMarshal.Error(), "fatal")
	}

	responseBody := bytes.NewBuffer(bodyJSON)

	// Send metadata+logs to user provided url
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequest("POST", pushURL, responseBody)
	if err != nil {
		spinner.LogMessage("Invalid push url: "+err.Error(), "fatal")
	}
	req.Header.Add("Content-Type", "application/json")
	sendPush(client, req, "build metadata and logs")
	spinner.LogMessage("Metadata+logs data pushed to provided url", "info")

	// Send artifact(s) to user provided url
	artifactList := os.Getenv("BUILDER_ARTIFACT_LIST")
	artifactArray := strings.Split(artifactList, ",")

	buf := bytes.NewBuffer(nil)
	m := multipart.NewWriter(buf)

	for i, artifact := range artifactArray {
		part, err := m.CreateFormFile("artifact"+fmt.Sprint(i), artifact)
		if err != nil {
			spinner.LogMessage("Error posting artifact(s) to url: "+err.Error(), "fatal")
		}

		file, err := os.Open(artifactDir + "/" + artifact)
		if err != nil {
			spinner.LogMessage("Error posting artifact(s) to url. Couldn't open artifact file: "+err.Error(), "fatal")
		}
		_, err = io.Copy(part, file)
		file.Close()
		if err != nil {
			spinner.LogMessage("Error reading artifact "+artifact+": "+err.Error(), "fatal")
		}
	}
	m.Close()

	artifactReq, err := http.NewRequest("POST", pushURL, buf)
	if err != nil {
		spinner.LogMessage("Invalid push url: "+err.Error(), "fatal")
	}
	artifactReq.Header.Add("Content-Type", m.FormDataContentType())
	sendPush(client, artifactReq, "artifact(s)")

	spinner.LogMessage("artifact(s) pushed to provided url", "info")
}

// sendPush sends req and fails the build on a transport error or a non-2xx response.
func sendPush(client *http.Client, req *http.Request, what string) {
	resp, err := client.Do(req)
	if err != nil {
		spinner.LogMessage("Error sending "+what+" to push url: "+err.Error(), "fatal")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		spinner.LogMessage("Push url rejected "+what+": "+resp.Status+": "+strings.TrimSpace(string(body)), "fatal")
	}
}

func getLogsJSON(path string) string {
	logsFile, err := os.Open(path)
	if err != nil {
		spinner.LogMessage("Can't find / open log file: "+err.Error(), "fatal")
	}
	defer logsFile.Close()

	var lines []string
	scanner := bufio.NewScanner(logsFile)
	scanner.Buffer(make([]byte, 0, 64*1024), MaxLogLine)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Re-format zap logger output to valid JSON
	asString := strings.Join(lines, ",")

	jsonString := "[" + asString + "]"

	return jsonString
}
