package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tkanos/gonfig"
)

// Configuration of Repository Service
type Configuration struct {
	Port              int
	RepositoryPath    string
	DefaultRepository string
}

var configuration Configuration

var banner string

func init() {
	banner = `
	_____                                   
	|     |___ _ _ ___ ___                   
	| | | | .'| | | -_|   |                  
	|_|_|_|__,|\_/|___|_|_|                  
	 _____                 _ _               
	| __  |___ ___ ___ ___|_| |_ ___ ___ _ _ 
	|    -| -_| . | . |_ -| |  _| . |  _| | |
	|__|__|___|  _|___|___|_|_| |___|_| |_  |
		  |_|                       |___|
`
}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())

	configuration = Configuration{}
	err := gonfig.GetConf("config/config.json", &configuration)
	if err != nil {
		log.Fatalf("error %#v", err)
	}

	e.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, fmt.Sprintf("<pre>%s</pre>", banner))
	})

	e.GET("/*", getArtifact)
	e.HEAD("/*", headArtifact)
	e.PUT("/*", putArtifact)

	e.GET("/repositories/:repositoryId/*", getArtifact)
	e.HEAD("/repositories/:repositoryId/*", headArtifact)
	e.PUT("/repositories/:repositoryId/*", putArtifact)

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

// An ArtifactFile is a struct representing the Artifact File itself.
type ArtifactFile struct {
	Repository string
	Name       string
	Path       string
	Location   string
	Folder     string
	Artifact   Artifact
}

func headArtifact(c echo.Context) error {
	setDefaultHeaders(c)
	artifactFile := mapArtifactFile(c)
	log.Printf("artifactFile: %#v", artifactFile)
	if _, err := os.Stat(artifactFile.Location); os.IsNotExist(err) {
		return c.String(http.StatusNotFound, fmt.Sprintf("%s does not exists!", artifactFile.Location))
	}
	c.Response().Header().Set("Content-Type", "application/octet-stream")
	fi, err := os.Stat(artifactFile.Location)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%#v", err))
	}
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	c.Response().Header().Set("Last-Modified", fi.ModTime().Format(http.TimeFormat))
	c.Response().Header().Set("ETag", getETag(artifactFile.Location))
	return c.String(http.StatusOK, fmt.Sprintf("%#v", artifactFile.Artifact))
}

func getArtifact(c echo.Context) error {
	setDefaultHeaders(c)
	artifactFile := mapArtifactFile(c)

	if artifactFile.Path == "" {
		return c.String(http.StatusNotFound, fmt.Sprintf("%s does not exists!", artifactFile.Location))
	}

	if _, err := os.Stat(artifactFile.Location); os.IsNotExist(err) {
		return c.String(http.StatusNotFound, fmt.Sprintf("%s does not exists!", artifactFile.Location))
	}

	log.Printf("\nrepository: %s\npath: %s\nartifact: %s\n",
		artifactFile.Repository,
		artifactFile.Path,
		fmt.Sprintf("%#v", artifactFile.Artifact))
	return c.File(artifactFile.Location)
}

func putArtifact(c echo.Context) error {
	setDefaultHeaders(c)
	artifactFile := mapArtifactFile(c)
	log.Printf("Checking path %s", artifactFile.Folder)
	err := os.MkdirAll(artifactFile.Folder, os.ModePerm)
	if err != nil {
		log.Printf("MkdirAll: %s", err.Error())
		return c.String(http.StatusInternalServerError, fmt.Sprintf("MkdirAll: %s", err.Error()))
	}
	outFile, err := os.Create(artifactFile.Location)
	if err != nil {
		log.Printf("os.Create: %s", err.Error())
		return c.String(http.StatusInternalServerError, fmt.Sprintf("os.Create: %s", err.Error()))
	}
	defer outFile.Close()
	n, err := io.Copy(outFile, c.Request().Body)
	if err != nil {
		log.Printf("io.Copy: %s", err.Error())
		return c.String(http.StatusInternalServerError, fmt.Sprintf("io.Copy: %s", err.Error()))
	}

	if strings.HasSuffix(strings.ToLower(artifactFile.Artifact.Version), "snapshot") {
		cleanArtifactSnapshot(artifactFile)
	}

	log.Printf("Writing %d bytes to %s", n, artifactFile.Location)
	return c.String(http.StatusOK, fmt.Sprintf("%#v", artifactFile.Artifact))
}

func getArtifactPath(artifact *Artifact) string {
	pathSeparator := string(os.PathSeparator)
	return path.Join(strings.Replace(artifact.GroupID, ".", pathSeparator, -1),
		artifact.ArtifactID,
		artifact.Version,
		artifact.File)
}

func cleanArtifactSnapshot(artifactFile *ArtifactFile) {
	log.Printf("cleaning %q", artifactFile)
	err := filepath.Walk(artifactFile.Folder, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, artifactFile.Artifact.Packaging) {
			//log.Printf("%s", path)
		}
		return nil
	})
	if err != nil {
		log.Printf("filepath.Walk: %s", err.Error())
	}
}

func mapArtifactFile(c echo.Context) *ArtifactFile {
	requestPath := c.Request().URL.Path
	repository := c.Param("repositoryId")
	if repository == "" {
		repository = configuration.DefaultRepository
	}
	artifactString := strings.Replace(requestPath, "/repositories/"+repository+"/", "", 1)
	artifact := mapArtifact(artifactString)
	artifactPath := getArtifactPath(artifact)
	artifactLocation := path.Join(configuration.RepositoryPath, repository, artifactPath)
	return &ArtifactFile{
		Repository: repository,
		Name:       artifactString,
		Path:       artifactPath,
		Location:   artifactLocation,
		Folder:     filepath.Dir(artifactLocation),
		Artifact:   *artifact}
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
	if partsLen-offset-3 < 0 {
		return &Artifact{}
	}
	version := parts[partsLen-2-offset]
	artifactID := parts[partsLen-3-offset]
	artifactFile := parts[len(parts)-1]
	artifactParts := strings.Split(artifactFile, ".")
	packaging := artifactParts[len(artifactParts)-1]
	groupdID := strings.Join(parts[:partsLen-3-offset], ".")
	classifier := mapClassifier(artifactString, packaging)
	log.Printf("Artifact: %s:%s:%s:%s:%s", groupdID, artifactID, classifier, packaging, version)
	return &Artifact{ArtifactID: artifactID, GroupID: groupdID, Version: version, Classifier: classifier, Packaging: packaging, File: artifactFile}
}

func mapClassifier(artifactString string, packaging string) string {
	parts := strings.Split(artifactString, "-")
	lastPart := parts[len(parts)-1]
	if len(parts) > 1 && strings.HasSuffix(lastPart, packaging) {
		if strings.Count(lastPart, ".") == 1 {
			return strings.Replace(lastPart, fmt.Sprintf(".%s", packaging), "", -1)
		}
	}
	return ""
}

func setDefaultHeaders(c echo.Context) {
	c.Response().Header().Set("Server", "repository/0.0.1")
}

func getETag(filePath string) string {
	var eTag string
	if _, err := os.Stat(filePath + ".sha1"); os.IsNotExist(err) {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Fatalf("%s does not exists!", filePath)
			return ""
		}
		b, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatalf("error %#v", err)
			return ""
		}
		eTag = fmt.Sprintf("%x", sha1.Sum(b))
	} else {
		b, err := ioutil.ReadFile(filePath + ".sha1")
		if err != nil {
			log.Fatalf("error %#v", err)
			return ""
		}
		eTag = string(b)
	}
	log.Printf("eTag of %s is %s", filePath, eTag)
	return eTag
}
