package version

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/apex/log"
	"net/http"
	"strings"
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

func CheckForUpdate(versionChannel chan *semver.Version) {
	log.Infof("Checking for newer version of FSOC")
	newestVersion, err := GetLatestVersion()
	log.WithField("Latest FSOC version", newestVersion).Infof("Latest fsoc version available: %s", newestVersion)
	if err != nil {
		log.Warnf(err.Error())
	}
	newestVersionSemVar, err := semver.NewVersion(newestVersion)
	if err != nil {
		log.WithField("unparseable version", newestVersion).Warnf("Could not parse version string: %w", err.Error())
	}
	versionChannel <- newestVersionSemVar
}

func CompareAndLogVersions(newestVersionSemVar *semver.Version) {
	currentVersion := GetVersion()
	currentVersionSemVer := ConvertVerToSemVar(currentVersion)
	newerVersionAvailable := currentVersionSemVer.Compare(newestVersionSemVar) == -1
	var debugFields = log.Fields{"IsLatestVersionDifferent": newerVersionAvailable, "CurrentVersion": currentVersionSemVer.String(), "LatestVersion": newestVersionSemVar.String()}

	if newerVersionAvailable {
		log.WithFields(debugFields).Warnf("There is a newer version of FSOC available, please upgrade from version")
	} else {
		log.WithFields(debugFields)
	}

}

func ConvertVerToSemVar(data VersionData) *semver.Version {
	return semver.New(
		uint64(data.VersionMajor),
		uint64(data.VersionMajor),
		uint64(data.VersionMajor),
		data.VersionMeta, "")
}
