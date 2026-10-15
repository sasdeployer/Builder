package yaml

import (
	"Builder/utils"
	"fmt"
	"os"
	"runtime"
	"strings"
)

func ConfigEnvs(byi interface{}) {
	//change interface{} into map interface{}
	bldyml, _ := byi.(map[string]interface{})

	//~~~Check for specific key and create env var based on value~~~

	//check for dir name
	if val, ok := bldyml["projectname"]; ok {
		_, present := os.LookupEnv("BUILDER_DIR_NAME")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILDER_DIR_NAME", valStr)
		}
	} else {
		os.Setenv("BUILDER_DIR_NAME", "")
	}

	//check for dir path
	if val, ok := bldyml["projectpath"]; ok {
		_, present := os.LookupEnv("BUILDER_DIR_PATH")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)

			// If on windows and they specify a path that begins with '/' append to home dir
			if runtime.GOOS == "windows" && strings.HasPrefix(valStr, "/") {
				homeDir := utils.GetUserData().HomeDir
				os.Setenv("BUILDER_DIR_PATH", homeDir+valStr)
			} else {
				os.Setenv("BUILDER_DIR_PATH", valStr)
			}
		}
	} else {
		os.Setenv("BUILDER_DIR_PATH", "")
	}

	//check for project type
	if val, ok := bldyml["projecttype"]; ok {
		_, present := os.LookupEnv("BUILDER_PROJECT_TYPE")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILDER_PROJECT_TYPE", valStr)
		}
	} else {
		os.Setenv("BUILDER_PROJECT_TYPE", "")
	}

	//check for different dir name to store builds
	if val, ok := bldyml["buildsdir"]; ok {
		_, present := os.LookupEnv("BUILDER_BUILDS_DIR")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILDER_BUILDS_DIR", valStr)
		}
	} else {
		os.Setenv("BUILDER_BUILDS_DIR", "")
	}

	//check for build type
	if val, ok := bldyml["buildtool"]; ok {
		_, present := os.LookupEnv("BUILDER_BUILD_TOOL")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILDER_BUILD_TOOL", valStr)
		}
	}

	//check for build file
	if val, ok := bldyml["buildfile"]; ok {
		_, present := os.LookupEnv("BUILDER_BUILD_FILE")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILDER_BUILD_FILE", valStr)
		}
	} else {
		os.Setenv("BUILDER_BUILD_FILE", "")
	}

	//check for config cmd
	if val, ok := bldyml["prebuildcmd"]; ok {
		_, present := os.LookupEnv("BUILDER_PREBUILD_COMMAND")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILDER_PREBUILD_COMMAND", valStr)
		}
	}

	//check for config cmd
	if val, ok := bldyml["configcmd"]; ok {
		_, present := os.LookupEnv("BUILDER_CONFIG_COMMAND")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILDER_CONFIG_COMMAND", valStr)
		}
	}

	//check for build cmd
	if val, ok := bldyml["buildcmd"]; ok {
		_, present := os.LookupEnv("BUILDER_BUILD_COMMAND")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILDER_BUILD_COMMAND", valStr)
		}
	}

	//check for output path
	if val, ok := bldyml["outputpath"]; ok {
		_, present := os.LookupEnv("BUILDER_OUTPUT_PATH")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)

			// If on windows and they specify a path that begins with '/' append to home dir
			if runtime.GOOS == "windows" && strings.HasPrefix(valStr, "/") {
				homeDir := utils.GetUserData().HomeDir
				os.Setenv("BUILDER_OUTPUT_PATH", homeDir+valStr)
			} else {
				os.Setenv("BUILDER_OUTPUT_PATH", valStr)
			}
		}
	}

	//check for docker cmd
	if val, ok := bldyml["dockercmd"]; ok {
		_, present := os.LookupEnv("BUILDER_DOCKER_CMD")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILDER_DOCKER_CMD", valStr)
		}
	}

	//check for global logs path
	if val, ok := bldyml["giturl"]; ok {
		_, present := os.LookupEnv("GIT_URL")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("GIT_URL", valStr)
		}
	}

	//check for an artifacts list
	if val, ok := bldyml["artifactlist"]; ok {
		_, present := os.LookupEnv("BUILDER_ARTIFACT_LIST")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILDER_ARTIFACT_LIST", valStr)
		}
	}

	//check for branch repo
	if val, ok := bldyml["repobranch"]; ok {
		_, present := os.LookupEnv("REPO_BRANCH")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("REPO_BRANCH", valStr)
		}
	}

	//check for options to build docker image
	if docker := subMap(bldyml["docker"]); docker != nil {
		if v := scalar(docker["dockerfile"]); v != "" {
			os.Setenv("BUILDER_DOCKERFILE", v)
		}
		if v := scalar(docker["registry"]); v != "" {
			os.Setenv("BUILDER_DOCKER_REGISTRY", v)
		}
		if v := scalar(docker["version"]); v != "" {
			os.Setenv("BUILDER_DOCKER_VERSION", v)
		}
	}

	//check for options to push resulting build data
	if push := subMap(bldyml["push"]); push != nil {
		if v := scalar(push["url"]); v != "" {
			os.Setenv("BUILDER_PUSH_URL", v)
		}
		if push["auto"] != nil {
			os.Setenv("BUILDER_PUSH_AUTO", scalar(push["auto"]))
		}
	}

	//check for app icon for build
	if val, ok := bldyml["appicon"]; ok {
		_, present := os.LookupEnv("BUILD_APP_ICON")
		if !present {
			//convert val interface{} to string to be set as env var
			valStr := fmt.Sprintf("%v", val)
			os.Setenv("BUILD_APP_ICON", valStr)
		}
	}
}

// subMap accepts either `key: {a: b}` or the list form `key: [{a: b}]` and
// returns nil for anything else, instead of panicking on unexpected yaml.
func subMap(v interface{}) map[string]interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		return t
	case []interface{}:
		if len(t) > 0 {
			if m, ok := t[0].(map[string]interface{}); ok {
				return m
			}
		}
	}
	return nil
}

// scalar renders a yaml scalar (string, number, bool) as a string; nil is "".
func scalar(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
