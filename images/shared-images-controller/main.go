package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.SetOutput(os.Stdout)
}

func getLatestTag() (tag string, err error) {
	resp, err := http.Get("https://raw.githubusercontent.com/kubevirt/kubevirt/main/cluster-up/version.txt")
	if err != nil {
		log.WithError(err).Errorf("Failed to get latest kubevirtci tag")
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
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
	context := context.Background()
	releases, _, err := client.Repositories.ListReleases(context, "kubernetes", "kubernetes", nil)
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
	kubevirtci_repo := "quay.io/kubevirtci/"

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

		name := fmt.Sprintf("%sk8s-%s:%s", kubevirtci_repo, version, tag)
		// k8s-1.26 kubevirtci provider was renamed to k8s-1.26-centos9
		// to track the change to CentOS Stream 9
		if version == "1.26" {
			name = fmt.Sprintf("%sk8s-%s-centos9:%s", kubevirtci_repo, version, tag)
		}
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

	var period time.Duration = 6

	socket := "unix:/run/podman/podman.sock"

	if os.Getenv("XDG_RUNTIME_DIR") != "" {
		sock_dir := os.Getenv("XDG_RUNTIME_DIR")
		socket = "unix:" + sock_dir + "/podman/podman.sock"
	}
	connText, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		log.WithError(err).Fatalf("Could not connect to podman socket %d", socket)
	}
	for {
		tag, err := getLatestTag()
		if err != nil {
			log.WithError(err).Errorf("Failed to get latest kubevirtci tag")
		}
		err = pullRequiredImages(connText, tag)
		if err != nil {
			log.WithError(err).Errorf("Failed to fetch image names")
		}
		err = cleanOldImages(connText, tag)
		if err != nil {
			log.WithError(err).Errorf("Failure occured when deleting old images")
		}
		log.Infof("Waiting for %d hours before checking again", period)
		time.Sleep(period * time.Hour)

	}
}
