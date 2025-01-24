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
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
	"gopkg.in/yaml.v2"
	"kubevirt.io/project-infra/robots/pkg/cannier"
	"os"
)

const defaultModelFilepath = "/tmp/kubevirt-cannier-model-data.yaml"

type RequestData struct {
	Features *cannier.FeatureSet `json:"features"`
}

type ResponseData struct {
	Classes     []float64 `json:"classes"`
	Description string    `json:"description"`
}

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

type StatValue struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

func newStat(name string, value float64) StatValue {
	return StatValue{
		Name:  name,
		Value: value,
	}
}

type StatCollection struct {
	Name  string      `json:"name"`
	Stats []StatValue `json:"stats"`
}

type Stats struct {
	FeatureStats []StatCollection
	ClassStats   StatCollection
}

func (d *ModelData) Stats() Stats {
	featureNames := cannier.FeatureNames()
	featureValues := make([][]float64, len(featureNames))
	for i := 0; i < len(featureNames); i++ {
		featureValues[i] = make([]float64, len(d.XData))
	}
	for i, featureVector := range d.XData {
		for k := range featureNames {
			featureValues[k][i] = featureVector[k]
		}
	}
	featureStats := make([]StatCollection, len(featureNames))
	for i, featureName := range featureNames {
		featureStats[i] = StatCollection{
			Name: featureName,
			Stats: []StatValue{
				newStat("Min", floats.Min(featureValues[i])),
				newStat("Max", floats.Max(featureValues[i])),
				newStat("Mean", stat.Mean(featureValues[i], nil)),
				newStat("StdDev", stat.StdDev(featureValues[i], nil)),
				newStat("Variance", stat.Variance(featureValues[i], nil)),
				newStat("Entropy", stat.Entropy(featureValues[i])),
				newStat("Count", float64(len(featureValues[i]))),
			},
		}
	}
	yDataFloats := make([]float64, len(d.YData))
	perLabelCounts := make(map[cannier.TestLabel]int, 3)
	for i, clz := range d.YData {
		yDataFloats[i] = float64(clz)
		perLabelCounts[cannier.TestLabel(clz)]++
	}
	classStats := StatCollection{
		Name: "TestLabel",
		Stats: []StatValue{
			newStat("Count", float64(len(d.YData))),
		},
	}
	testLabels := cannier.TestLabels()
	for clz, perLabelCount := range perLabelCounts {
		classStats.Stats = append(classStats.Stats, newStat(fmt.Sprintf("Count_%s", testLabels[clz]), float64(perLabelCount)))
	}
	return Stats{featureStats, classStats}
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
