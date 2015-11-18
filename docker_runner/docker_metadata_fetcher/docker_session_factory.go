package docker_metadata_fetcher

import (
	"fmt"
	"net/http"

	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/registry"
)

//go:generate counterfeiter -o fake_docker_session/fake_docker_session.go . DockerSession
type DockerSession interface {
	GetRepositoryData(remote string) (*registry.RepositoryData, error)
	GetRemoteTags(registries []string, repository string) (map[string]string, error)
	GetRemoteImageJSON(imgID, registry string) ([]byte, int, error)
}

//go:generate counterfeiter -o fake_docker_session/fake_docker_session_factory.go . DockerSessionFactory
type DockerSessionFactory interface {
	MakeSession(reposName string, allowInsecure bool) (DockerSession, error)
}

type dockerSessionFactory struct{}

func NewDockerSessionFactory() *dockerSessionFactory {
	return &dockerSessionFactory{}
}

func (factory *dockerSessionFactory) MakeSession(reposName string, allowInsecure bool) (DockerSession, error) {
	repositoryInfo, err := registry.ParseRepositoryInfo(reposName)
	if err != nil {
		return nil, fmt.Errorf("Error resolving Docker repository name:\n" + err.Error())
	}

	if allowInsecure {
		repositoryInfo.Index.Secure = false
	}
	endpoint, err := registry.NewEndpoint(repositoryInfo.Index, nil, registry.APIVersionUnknown)
	if err != nil {
		return nil, fmt.Errorf("Error Connecting to Docker registry:\n" + err.Error())
	}
	authConfig := &cliconfig.AuthConfig{}
	session, error := registry.NewSession(http.DefaultClient, authConfig, endpoint)
	return session, error
}
