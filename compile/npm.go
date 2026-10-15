package compile

import (
	"Builder/artifact"
	"Builder/directory"
	"Builder/spinner"
	"Builder/utils"
	"Builder/utils/log"
	"Builder/yaml"
	"bufio"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	cp "github.com/otiai10/copy"
)

// Npm creates zip from files passed in as arg
func Npm() {
	//Set default project type env for builder.yaml creation
	projectType := os.Getenv("BUILDER_PROJECT_TYPE")
	if projectType == "" {
		os.Setenv("BUILDER_PROJECT_TYPE", "node")
	}

	//Set up local logger
	localPath, _ := os.LookupEnv("BUILDER_LOGS_DIR")
	locallogger, closeLocalLogger = log.NewLogger("logs", localPath)

	hiddenDir := os.Getenv("BUILDER_HIDDEN_DIR")
	workspaceDir := os.Getenv("BUILDER_WORKSPACE_DIR")
	tempWorkspace := workspaceDir + "/temp/"
	//make temp dir
	os.Mkdir(tempWorkspace, 0755)

	//add hidden dir contents to temp dir, install dependencies
	if err := utils.CloneTree(hiddenDir, tempWorkspace, nil); err != nil {
		spinner.LogMessage(err.Error(), "warn")
	}

	//define dir path for command to run
	var fullPath string
	configPath := os.Getenv("BUILDER_DIR_PATH")
	//if user defined path in builder.yaml, full path is included in tempWorkspace, else add the local path
	if os.Getenv("BUILDER_COMMAND") == "true" {
		// ex: C:/Users/Name/Projects/helloworld_19293/workspace/dir
		fullPath = tempWorkspace
	} else if configPath != "" {
		fullPath = tempWorkspace
	} else {
		path, _ := os.Getwd()
		//combine local path to newly created tempWorkspace, gets rid of "." in path name
		fullPath = path + tempWorkspace[strings.Index(tempWorkspace, ".")+1:]
		os.Setenv("BUILDER_DIR_PATH", path)
	}

	//install dependencies/build, if yaml build type exists install accordingly
	buildTool := strings.ToLower(os.Getenv("BUILDER_BUILD_TOOL"))
	buildCmd := os.Getenv("BUILDER_BUILD_COMMAND")

	var cmd *exec.Cmd
	if buildCmd != "" {
		//user specified cmd
		buildCmdArray := strings.Fields(buildCmd)
		cmd = exec.Command(buildCmdArray[0], buildCmdArray[1:]...)
		cmd.Dir = fullPath // or whatever directory it's in
	} else {
		// default: install exactly what the lockfile pins
		tool, args := nodeInstall(fullPath, buildTool)
		cmd = exec.Command(args[0], args[1:]...)
		cmd.Dir = fullPath // or whatever directory it's in
		os.Setenv("BUILDER_BUILD_TOOL", tool)
		os.Setenv("BUILDER_BUILD_COMMAND", strings.Join(args, " "))
	}

	//run cmd, check for err, log cmd
	spinner.LogMessage("running command: "+cmd.String(), "info")

	stdout, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		spinner.LogMessage(pipeErr.Error(), "fatal")
	}

	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), utils.MaxLogLine)

	// Use the scanner to scan the output line by line and log it
	// It's running in a goroutine so that it doesn't block
	go func() {
		// Read line by line and process it
		for scanner.Scan() {
			line := scanner.Text()
			spinner.Spinner.Stop()
			locallogger.Info(line)
			spinner.Spinner.Start()
		}
		if err := scanner.Err(); err != nil {
			locallogger.Warn("build output capture stopped: " + err.Error())
			io.Copy(io.Discard, stdout)
		}

		// We're all done, unblock the channel
		done <- struct{}{}

	}()

	os.Setenv("BUILD_START_TIME", time.Now().Format(time.RFC850))

	if err := cmd.Start(); err != nil {
		spinner.LogMessage(err.Error(), "fatal")
	}

	// Wait for all output to be processed
	<-done

	// Wait for cmd to finish
	if err := cmd.Wait(); err != nil {
		spinner.LogMessage(err.Error(), "fatal")
	}

	os.Setenv("BUILD_END_TIME", time.Now().Format(time.RFC850))

	// Close log file
	closeLocalLogger()

	// Update parent dir name to include start time and send back new full path
	fullPath = directory.UpdateParentDirName(fullPath)

	// Update vars because of parent dir name change
	workspaceDir = os.Getenv("BUILDER_WORKSPACE_DIR")
	tempWorkspace = workspaceDir + "/temp/"

	if len(os.Args) > 1 && (os.Args[1] == "init" || os.Args[1] == "config") {
		repoPath := "./" + strings.TrimSuffix(utils.GetName(), ".git")

		if configPath != "" {
			repoPath = configPath + "/" + strings.TrimSuffix(utils.GetName(), ".git")
		}

		yaml.CreateBuilderYaml(repoPath)
	} else {
		yaml.CreateBuilderYaml(fullPath)
	}

	// CreateZip artifact dir with timestamp
	parsedStartTime, _ := time.Parse(time.RFC850, os.Getenv("BUILD_START_TIME"))
	timeBuildStarted := parsedStartTime.Unix()

	if err := zipTree(workspaceDir+"/artifact_"+strconv.FormatInt(timeBuildStarted, 10)+".zip", tempWorkspace); err != nil {
		spinner.LogMessage("node-npm project failed to compile: "+err.Error(), "fatal")
	}

	packageNpmArtifact(fullPath)
	// artifactPath := os.Getenv("BUILDER_OUTPUT_PATH")
	// fmt.Print(artifactPath)
	// if artifactPath != "" {
	// 	artifactZip := os.Getenv("BUILDER_ARTIFACT_STAMP")
	// 	fmt.Print(artifactZip)
	// 	exec.Command("cp", "-a", artifactZip+".zip", artifactPath).Run()
	// }
	spinner.LogMessage("node-npm project compiled successfully", "info")
}

func packageNpmArtifact(fullPath string) {
	// archiveExt := ""

	// if runtime.GOOS == "windows" {
	// 	archiveExt = ".zip"
	// } else {
	// 	archiveExt = ".tar.gz"
	// }

	artifact.ArtifactDir()
	artifactDir := os.Getenv("BUILDER_ARTIFACT_DIR")
	workspaceDir := os.Getenv("BUILDER_WORKSPACE_DIR")
	outputPath := os.Getenv("BUILDER_OUTPUT_PATH")

	//find artifact by extension
	found, extName := artifact.ExtExistsFunction(workspaceDir, ".zip")
	if !found {
		spinner.LogMessage("Build finished but produced no artifact (expected the dependency zip in "+workspaceDir+").  Check the build command, or set artifactlist in builder.yaml", "fatal")
	}
	os.Setenv("BUILDER_ARTIFACT_NAMES", extName)

	//copy artifact, then remove artifact in workspace
	if err := moveFile(workspaceDir+"/"+extName, artifactDir+"/"+extName); err != nil {
		spinner.LogMessage("could not move artifact into the artifact dir: "+err.Error(), "fatal")
	}

	// If outputpath provided also cp artifacts to that location
	if outputPath != "" {
		// Check if outputPath exists.  If not, create it
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			if err := os.Mkdir(outputPath, 0755); err != nil {
				spinner.LogMessage("Could not create output path", "fatal")
			}
		}

		err := cp.Copy(artifactDir+"/"+extName, outputPath+"/"+extName)
		if err != nil {
			spinner.LogMessage(err.Error(), "warn")
		}

		spinner.LogMessage("Artifact(s) copied to output path provided", "info")
	}

	//create metadata, then copy contents to zip dir
	utils.Metadata(artifactDir)

	// if os.Getenv("ARTIFACT_ZIP_ENABLED") == "true" {
	// 	//zip artifact
	// 	artifact.ZipArtifactDir()

	// 	//copy zip into open artifactDir, delete zip in workspace (keeps entire artifact contained)
	// 	exec.Command("cp", "-a", artifactDir+archiveExt, artifactDir).Run()
	// 	exec.Command("rm", artifactDir+archiveExt).Run()

	// 	// artifactName := artifact.NameArtifact(fullPath, extName)

	// 	// send artifact to user specified path or send to parent directory
	// 	artifactStamp := os.Getenv("BUILDER_ARTIFACT_STAMP")
	// 	outputPath := os.Getenv("BUILDER_OUTPUT_PATH")
	// 	if outputPath != "" {
	// 		exec.Command("cp", "-a", artifactDir+"/"+artifactStamp+archiveExt, outputPath).Run()
	// 	} else {
	// 		exec.Command("cp", "-a", artifactDir+"/"+artifactStamp+archiveExt, os.Getenv("BUILDER_PARENT_DIR")).Run()
	// 	}

	// 	//remove artifact directory
	// 	exec.Command("rm", "-r", artifactDir).Run()
	// }
}

// nodeInstall picks the install command for the lockfile in dir, so installs
// are reproducible: npm ci, or --frozen-lockfile for pnpm/yarn/bun. Without a
// lockfile it falls back to npm install (and the build isn't cached).
func nodeInstall(dir, buildTool string) (string, []string) {
	exists := func(name string) bool {
		_, err := os.Stat(dir + "/" + name)
		return err == nil
	}
	switch {
	case exists("package-lock.json") || exists("npm-shrinkwrap.json"):
		return "npm", []string{"npm", "ci"}
	case buildTool == "npm":
		return "npm", []string{"npm", "install"}
	case exists("pnpm-lock.yaml"):
		return "pnpm", []string{"pnpm", "install", "--frozen-lockfile"}
	case exists("yarn.lock"):
		return "yarn", []string{"yarn", "install", "--frozen-lockfile"}
	case exists("bun.lock") || exists("bun.lockb"):
		return "bun", []string{"bun", "install", "--frozen-lockfile"}
	}
	return "npm", []string{"npm", "install"}
}
