package version

import (
	"fmt"
	"io"
	"net/http"
	"os"

	config_package "github.com/cloudfoundry-incubator/ltc/config"
	"github.com/cloudfoundry-incubator/receptor"
)

//go:generate counterfeiter -o fake_file_swapper/fake_file_swapper.go . FileSwapper
type FileSwapper interface {
	GetTempFile() (*os.File, error)
	SwapTempFile(destPath, srcPath string) error
}

type ServerVersions struct {
	CFRelease           string
	CFRoutingRelease    string
	DiegoRelease        string
	GardenLinuxRelease  string
	LatticeRelease      string
	LatticeReleaseImage string
	LTC                 string
	Receptor            string
}

//go:generate counterfeiter -o fake_version_manager/fake_version_manager.go . VersionManager
type VersionManager interface {
	SyncLTC(ltcPath string, arch string, config *config_package.Config) error
	ServerVersions() (ServerVersions, error)
	LatticeVersion() string
	LtcMatchesServer() (bool, error)
}

type versionManager struct {
	receptorClient receptor.Client
	fileSwapper    FileSwapper
	latticeVersion string
}

func NewVersionManager(receptorClient receptor.Client, fileSwapper FileSwapper, latticeVersion string) *versionManager {
	return &versionManager{
		receptorClient,
		fileSwapper,
		latticeVersion,
	}
}

func (v *versionManager) ServerVersions() (ServerVersions, error) {
	versionResponse, err := v.receptorClient.GetVersion()
	if err != nil {
		return ServerVersions{}, err
	}
	return ServerVersions{
		CFRelease:           versionResponse.CFRelease,
		CFRoutingRelease:    versionResponse.CFRoutingRelease,
		DiegoRelease:        versionResponse.DiegoRelease,
		GardenLinuxRelease:  versionResponse.GardenLinuxRelease,
		LatticeRelease:      versionResponse.LatticeRelease,
		LatticeReleaseImage: versionResponse.LatticeReleaseImage,
		LTC:                 versionResponse.LTC,
		Receptor:            versionResponse.Receptor,
	}, nil
}

func (s *versionManager) SyncLTC(ltcPath string, arch string, config *config_package.Config) error {
	response, err := http.DefaultClient.Get(fmt.Sprintf("%s/v1/sync/%s/ltc", config.Receptor(), arch))
	if err != nil {
		return fmt.Errorf("failed to connect to receptor: %s", err.Error())
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("failed to download ltc: %s", response.Status)
	}

	tmpFile, err := s.fileSwapper.GetTempFile()
	if err != nil {
		return fmt.Errorf("failed to open temp file: %s", err.Error())
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, response.Body)
	if err != nil {
		return fmt.Errorf("failed to write to temp ltc: %s", err.Error())
	}

	err = s.fileSwapper.SwapTempFile(ltcPath, tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to swap ltc: %s", err.Error())
	}

	return nil
}

func (s *versionManager) LatticeVersion() string {
	return s.latticeVersion
}

func (s *versionManager) LtcMatchesServer() (bool, error) {
	serverVersions, err := s.ServerVersions()
	if err != nil {
		return false, err
	}

	match := (serverVersions.LatticeRelease == s.LatticeVersion())
	return match, nil
}
