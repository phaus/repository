# Maven Repository Server in golang

The motivation to build this is basically this issue:

https://gitlab.com/gitlab-org/gitlab-ee/issues/2752

## an URL of a Maven Artifact is defined as:

    <repositoryUrl>/<groupID>/<artifactID>/<version>/<artifactID>-<version>.<type>

The `repositoryUrl` is defined here as `host:port`/repositories/`repositoryID`


You need to add a configuration in `config`. This might look like this:

````
{
    "Port":5000,
    "RepositoryPath": "/folder/.m2"
}
````