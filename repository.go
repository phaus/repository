package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"strings"

	"github.com/labstack/echo"
	"github.com/tkanos/gonfig"
)

// Configuration of Repository Service
type Configuration struct {
	Port           int
	RepositoryPath string
}

var configuration Configuration

func main() {
	e := echo.New()

	configuration = Configuration{}
	err := gonfig.GetConf("config/config.json", &configuration)
	if err != nil {
		log.Fatal(err)
	}

	e.GET("/repositories/:repositoryId/*", getArtifact)

	//	e.POST("/users", saveUser)

	//	e.PUT("/users/:id", updateUser)
	//	e.DELETE("/users/:id", deleteUser)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", configuration.Port)))
}

// An Artifact structure is used hold the details of a Maven Artifact.
type Artifact struct {
	GroupID    string
	ArtifactID string
	Version    string
	Classifier string
	Packaging  string
	File       string
}

func getArtifact(c echo.Context) error {
	path := c.Request().URL.Path
	repository := c.Param("repositoryId")
	artifactString := strings.Replace(path, "/repositories/"+repository+"/", "", 1)
	artifact := mapArtifact(artifactString)
	artifactPath := getArtifactPath(artifact)
	artifactLocation := fmt.Sprintf("%s%s%s%s%s", configuration.RepositoryPath, string(os.PathSeparator), repository, string(os.PathSeparator), artifactPath)
	if _, err := os.Stat(artifactLocation); os.IsNotExist(err) {
		fmt.Printf("%s does not exists!", artifactLocation)
		return c.String(http.StatusNotFound, fmt.Sprintf("%s does not exists!", artifactLocation))
	}
	fmt.Printf("\nrepository: %s\npath: %s\nartifact: %s\n",
		repository,
		artifactPath,
		fmt.Sprintf("%#v", artifact))
	return c.File(artifactLocation)
}

func getArtifactPath(artifact *Artifact) string {
	pathSeparator := string(os.PathSeparator)
	return fmt.Sprintf("%s%s%s%s%s%s%s",
		strings.Replace(artifact.GroupID, ".", pathSeparator, -1),
		pathSeparator,
		artifact.ArtifactID,
		pathSeparator,
		artifact.Version,
		pathSeparator,
		artifact.File)
}

func mapArtifact(artifactString string) *Artifact {
	parts := strings.Split(artifactString, "/")
	partsLen := len(parts)

	var offset int
	if "jars" == parts[partsLen-2] {
		offset = 1
	} else {
		offset = 0
	}
	version := parts[partsLen-2-offset]
	artifactID := parts[partsLen-3-offset]
	artifactFile := parts[len(parts)-1]
	artifactParts := strings.Split(artifactFile, ".")

	packaging := artifactParts[len(artifactParts)-1]
	groupdID := strings.Join(parts[:partsLen-3-offset], ".")
	fmt.Printf("%q %d\n", parts, partsLen)
	fmt.Printf("%q %d\n", artifactParts, len(artifactParts))
	return &Artifact{ArtifactID: artifactID, GroupID: groupdID, Version: version, Classifier: "", Packaging: packaging, File: artifactFile}
}
