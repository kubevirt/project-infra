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
 * Copyright the KubeVirt Authors.
 *
 */

package cmd

import (
	"fmt"
	randomforest "github.com/malaschitz/randomForest"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"kubevirt.io/project-infra/robots/pkg/flake-heuristic/cannier"
	"os"
)

type TestDataPool struct {
	TestDescriptor
	testFileName string
	Features     *cannier.FeatureSet
}

type ModelData struct {
	XData [][]float64 `yaml:"x_data"`
	YData []int       `yaml:"y_data"`
}

func (d *ModelData) Append(x []float64, y int) {
	d.XData = append(d.XData, x)
	d.YData = append(d.YData, int(y))
}

func (d *ModelData) Boruta() (importantFeatures []int, mapOfFeatures map[int]int) {
	return randomforest.BorutaDefault(d.XData, d.YData)
}

func (d *ModelData) Model() randomforest.Forest {
	forest := randomforest.Forest{}
	forest.Data = randomforest.ForestData{X: d.XData, Class: d.YData}
	return forest
}

func (d *ModelData) Store(dataFilepath string) error {
	file, err := os.Create(dataFilepath)
	if err != nil {
		return fmt.Errorf("could not create output file %q: %v", dataFilepath, err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Error(err)
		}
	}(file)
	err = yaml.NewEncoder(file).Encode(&d)
	if err != nil {
		return fmt.Errorf("could not write output file %q: %v", dataFilepath, err)
	}
	return nil
}

func Load(dataFilepath string) (*ModelData, error) {
	file, err := os.Open(dataFilepath)
	if err != nil {
		return nil, fmt.Errorf("could not open input file %q: %v", dataFilepath, err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Error(err)
		}
	}(file)
	var d *ModelData
	err = yaml.NewDecoder(file).Decode(&d)
	if err != nil {
		return nil, fmt.Errorf("could not read input file %q: %v", dataFilepath, err)
	}
	return d, nil

}
