package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

const (
	registryVersion = "2.8.2"
	checkInterval   = 6 * time.Hour
	kubevirtciRepo  = "quay.io/kubevirtci/"
)

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.SetOutput(os.Stdout)
}

func getLatestTag() (tag string, err error) {
	resp, err := http.Get("https://raw.githubusercontent.com/kubevirt/kubevirt/main/kubevirtci/cluster-up/version.txt")
	if err != nil {
		log.WithError(err).Errorf("Failed to get latest kubevirtci tag")
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Errorf("Reading latest kubevirtci tag failed")
		return "", err
	}
	tag = string(body)
	return strings.TrimSpace(tag), nil
}

func pullRequiredImages(ctx context.Context, tag string) error {
	var versions []string

	endpoint := "https://api.github.com/"
	client, err := github.NewEnterpriseClient(endpoint, endpoint, nil)
	if err != nil {
		log.WithError(err)
		return err
	}
	ghCtx := context.Background()
	releases, _, err := client.Repositories.ListReleases(ghCtx, "kubernetes", "kubernetes", nil)
	if err != nil {
		log.WithError(err).Errorf("Failed to list releases from kubernetes/kubernetes")
		return err
	}
	minors := querier.LastThreeMinor(uint(1), releases)
	for _, minor := range minors {
		r := querier.ParseRelease(minor)
		releaseVer := fmt.Sprintf("%s.%s", r.Major, r.Minor)
		versions = append(versions, releaseVer)
	}

	log.Infoln("Last three minor releases", versions)

	imageNames := map[string]struct{}{}

	imageList, err := images.List(ctx, nil)
	if err != nil {
		log.WithError(err).Errorf("Failed to list images")
		return err
	}
	for _, i := range imageList {
		for _, imageName := range i.RepoTags {
			imageNames[imageName] = struct{}{}
		}
	}

	for _, version := range versions {
		log.Infoln("Kubevirt Provider version: ", version)

		name := fmt.Sprintf("%sk8s-%s:%s", kubevirtciRepo, version, tag)
		if _, exists := imageNames[name]; exists {
			log.Infoln("Image already present:", name)
			continue
		}
		log.Infoln("Pulling image: ", name)
		_, err := images.Pull(ctx, name, nil)
		if err != nil {
			log.WithError(err).Errorf("Failed to pull image '%s'", name)
		}
	}

	for _, image := range []string{fmt.Sprintf("%sgocli:%s", kubevirtciRepo, tag), fmt.Sprintf("quay.io/libpod/registry:%s", registryVersion)} {
		_, err = images.Pull(ctx, image, nil)
		if err != nil {
			log.WithError(err).Errorf("Failed to pull image '%s'", image)
		}
	}

	return nil
}

func cleanOldImages(ctx context.Context, tag string) error {
	var deleteImages []string
	imageList, err := images.List(ctx, nil)
	if err != nil {
		log.WithError(err).Errorf("Failed to list images")
		return err
	}
	for _, i := range imageList {
		for _, repoTag := range i.RepoTags {
			if strings.Contains(repoTag, tag) {
				log.Infof("%s is a required image", repoTag)
				continue
			}
			if strings.Contains(repoTag, registryVersion) {
				log.Infof("%s is a required image", repoTag)
				continue
			}
			log.Infoln("Deleting image:", repoTag)
			deleteImages = append(deleteImages, repoTag)

		}
	}
	log.Infoln("images to delete:", deleteImages)
	if len(deleteImages) > 0 {
		_, err := images.Remove(ctx, deleteImages, nil)
		if err != nil {
			return err[0]
		}
	}
	return nil
}

func main() {
	socket := "unix:/run/podman/podman.sock"

	if os.Getenv("XDG_RUNTIME_DIR") != "" {
		sockDir := os.Getenv("XDG_RUNTIME_DIR")
		socket = "unix:" + sockDir + "/podman/podman.sock"
	}
	connText, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		log.WithError(err).Fatalf("Could not connect to podman socket %s", socket)
	}
	for {
		log.Infof("Waiting for %.0f hours before checking again", checkInterval.Hours())
		time.Sleep(checkInterval)

		tag, err := getLatestTag()
		if err != nil {
			log.WithError(err).Errorf("Failed to get latest kubevirtci tag")
			continue
		}
		err = pullRequiredImages(connText, tag)
		if err != nil {
			log.WithError(err).Errorf("Failed to fetch image names")
			continue
		}
		err = cleanOldImages(connText, tag)
		if err != nil {
			log.WithError(err).Errorf("Failure occurred when deleting old images")
		}
	}
}
