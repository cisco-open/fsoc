package version

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/apex/log"
)

const (
	versionLatestURL = "https://github.com/cisco-open/fsoc/releases/latest"
)

func GetLatestVersion() (string, error) {
	// Open HTTP client which will not follow redirect
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error { // no redirect
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(versionLatestURL)
	if err != nil {
		return "", err
	}
	// Redirected link should look like https://github.com/cisco-open/fsoc/releases/tag/{VERSION}
	split := strings.Split(resp.Header.Get("Location"), "/")
	if len(split) < 1 {
		return "", fmt.Errorf("version request did not return a version")
	}
	return split[len(split)-1], nil
}

func CheckForUpdate() *semver.Version {
	log.Infof("Checking for newer version of fsoc")
	newestVersion, err := GetLatestVersion()
	if err == nil {
		log.WithField("latest_github_version", newestVersion).Info("Latest fsoc version available")
	} else {
		log.Warnf("Failed to get latest fsoc version number from github: %v", err)
	}
	newestVersionSemVar, err := semver.NewVersion(newestVersion)
	if err != nil {
		log.WithField("version_tag", newestVersion).Warnf("Could not parse version tag as a semver: %v", err)
	}
	return newestVersionSemVar
}

func CompareAndLogVersions(newestVersionSemVar *semver.Version) {
	currentVersion := GetVersion()
	currentVersionSemVer := ConvertVerToSemVar(currentVersion)
	newerVersionAvailable := currentVersionSemVer.Compare(newestVersionSemVar) < 0
	var debugFields = log.Fields{"current_version": currentVersionSemVer.String(), "latest_version": newestVersionSemVar.String()}

	if IsDev() {
		log.WithFields(debugFields).Warnf("Running a local build of fsoc that may not have the latest improvements")
	} else if newerVersionAvailable {
		log.WithFields(debugFields).Warnf("There is a newer version of fsoc available, please upgrade")
	} else {
		debugFields["version_cmp"] = newerVersionAvailable
		log.WithFields(debugFields).Info("fsoc version check completed; no upgrade available")
	}

}

func ConvertVerToSemVar(data VersionData) *semver.Version {
	return semver.New(
		uint64(data.VersionMajor),
		uint64(data.VersionMinor),
		uint64(data.VersionPatch),
		data.VersionMeta, "")
}
