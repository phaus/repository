package main

import (
	"fmt"
	"log"
	"net/http"

	"strings"

	"github.com/labstack/echo"
	"github.com/tkanos/gonfig"
)

// Configuration of Repository Service
type Configuration struct {
	Port           int
	RepositoryPath string
}

func main() {
	e := echo.New()

	configuration := Configuration{}
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
}

func getArtifact(c echo.Context) error {
	path := c.Request().URL.Path
	repository := c.Param("repositoryId")
	artifactString := strings.Replace(path, "/repositories/"+repository+"/", "", 1)
	artifact := mapArtifact(artifactString)
	return c.String(http.StatusOK, "repository:"+repository+", artifact:"+fmt.Sprintf("%#v", artifact))
}

// com/nimbusds/nimbus-jose-jwt/3.9/nimbus-jose-jwt-3.9.pom
// groupId/artifactId/version/artifactId-version
func mapArtifact(artifactString string) *Artifact {
	parts := strings.Split(artifactString, "/")
	partsLen := len(parts)
	fmt.Printf("%q %d\n", parts, partsLen)
	var offset int
	if "jars" == parts[partsLen-2] {
		offset = 1
	} else {
		offset = 0
	}
	version := parts[partsLen-2-offset]
	artifactID := parts[partsLen-3-offset]
	artifactParts := strings.Split(parts[len(parts)-1], ".")
	fmt.Printf("%q %d\n", artifactParts, len(artifactParts))
	packaging := artifactParts[len(artifactParts)-1]
	groupdID := strings.Join(parts[:partsLen-3-offset], ".")

	return &Artifact{ArtifactID: artifactID, GroupID: groupdID, Version: version, Classifier: "", Packaging: packaging}
}
