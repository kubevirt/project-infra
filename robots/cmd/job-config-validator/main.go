/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"flag"
	"io/fs"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runc/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func flagOptions() options {
	o := options{}
	flag.StringVar(&o.configMapName, "config-map-name", "job-config", "The name of the config map that contains the job config files")
	flag.StringVar(&o.configMapNameSpace, "config-map-namespace", "kubevirt-prow", "The namespace of the config map that contains the job config files")
	flag.StringVar(&o.jobConfigPath, "job-config-path", "github/ci/prow-deploy/files/jobs/kubevirt/kubevirt", "The path to the job configuration files that need to be present")
	flag.Parse()
	return o
}

func (o *options) validate() error {
	_, err := os.Stat(o.jobConfigPath)
	if err == os.ErrNotExist {
		return err
	}

	return nil
}

type options struct {
	configMapName      string
	jobConfigPath      string
	configMapNameSpace string
}

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
	log := logrus.StandardLogger().WithField("robot", "job-config-validator")

	opts := flagOptions()
	err := opts.validate()
	if err != nil {
		log.Fatalf("Arguments invalid: %v", err)
	}

	clientset, err := NewClientset()
	configMap, err := clientset.CoreV1().ConfigMaps(opts.configMapNameSpace).Get(context.Background(), opts.configMapName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Failed to fetch configmap %q in namespace %q : %v", opts.configMapName, opts.configMapNameSpace, err)
	}

	// fetch list of job config files from all subdirectories, put them into a map with each filename as a key for
	// direct access
	jobConfigFileNames := map[string]struct{}{}
	err = filepath.WalkDir(opts.jobConfigPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		jobConfigFileNames[d.Name()] = struct{}{}
		return nil
	})
	if err != nil {
		log.Errorf("Failed to fetch job config names in dir %q: %v", opts.jobConfigPath, err)
	}

	// remove each file for which we find a key in the configMap from the list
	for key, _ := range configMap.Data {
		if _, exists := jobConfigFileNames[key]; exists {
			delete(jobConfigFileNames, key)
		}
	}
	// If ConfigMapSpec is GZIP compressed the keys will be found there
	for key, _ := range configMap.BinaryData {
		if _, exists := jobConfigFileNames[key]; exists {
			delete(jobConfigFileNames, key)
		}
	}

	// if there's anything left in the list, the job config is not complete
	if len(jobConfigFileNames) > 0 {
		log.Fatalf("Expected entries in config map %q missing: %v", opts.configMapName, jobConfigFileNames)
	}
}

func NewClientset() (*kubernetes.Clientset, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func GetConfig() (*restclient.Config, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = path.Join(usr.HomeDir, ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
