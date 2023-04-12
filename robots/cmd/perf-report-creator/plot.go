package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"github.com/MetalBlueberry/go-plotly/offline"
	gonumplot "gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"k8s.io/apimachinery/pkg/util/errors"
)

var DateFormat = "2006-01-02"

type Curve struct {
	X     []string
	Y     []float64
	Color color.RGBA
	Title string
}

type PlotData struct {
	Title      string
	XAxisLabel string
	YAxisLabel string
	Curves     []Curve
}

func gatherPlotData(basePath string, resource string, metric ResultType) ([]Curve, error) {
	totalCurves := 2
	curves := make([]Curve, totalCurves)
	for i := 0; i < totalCurves; i++ {
		curves[i].X = []string{}
		curves[i].Y = []float64{}
		if i == 0 {
			curves[i].Color = color.RGBA{0, 255, 0, 255}
		} else {
			curves[i].Color = color.RGBA{0, 0, 255, 255}
		}
	}

	err := filepath.Walk(filepath.Join(basePath, resource, string(metric)), func(entryPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//if !(info.IsDir()) {
		//	//fmt.Println(info.IsDir(), info.Name())
		//	return nil
		//}
		if entryPath == basePath {
			return nil
		}

		//const JSONResultsFileName = "results.json"
		if !info.IsDir() && strings.Contains(entryPath, "results.json") {
			fmt.Println(entryPath)
			jsonFile, err := os.Open(entryPath)
			if err != nil {
				return err
			}
			//defer jsonFile.Close()
			byteValue, err := ioutil.ReadAll(jsonFile)
			if err != nil {
				return err
			}

			var record Record
			err = json.Unmarshal(byteValue, &record)
			if err != nil {
				return err
			}

			//switch metricName {
			//case constants.MergeQueueLengthName:
			//	addPRRangeResults(results, curves, metricName)
			//case constants.RetestsToMergeName, constants.TimeToMergeName:
			//	addPRUnitResults(results, curves, metricName)
			//}
			addMetricRangeResults(record, curves)

			curves[1].X = append(curves[1].X, record.StartDate)
			curves[1].Y = append(curves[1].Y, record.Data.Average)

			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return curves, nil
}

func addMetricRangeResults(record Record, curves []Curve) {
	for _, dataPoint := range record.Data.DataPoints {
		dataPoint := dataPoint
		if dataPoint.Date != nil {
			curves[0].X = append(curves[0].X, dataPoint.Date.Format(DateFormat))
			curves[0].Y = append(curves[0].Y, dataPoint.Value)
		}
	}
}

func drawStaticGraph(filePath string, data PlotData) error {
	xticks := gonumplot.TimeTicks{Format: DateFormat}

	p := gonumplot.New()
	p.Title.Text = data.Title
	p.X.Tick.Marker = xticks
	p.X.Label.Text = data.XAxisLabel
	p.Y.Label.Text = data.YAxisLabel
	p.Add(plotter.NewGrid())

	// first curve represent raw data, second curve aggregates
	for cont, curve := range data.Curves {
		data, err := transformForStaticGraph(curve.X, curve.Y)
		if err != nil {
			return err
		}

		var line *plotter.Line
		var points *plotter.Scatter
		if cont == 0 {
			points, err = plotter.NewScatter(data)
			if err != nil {
				return err
			}
		} else {
			line, points, err = plotter.NewLinePoints(data)
			if err != nil {
				return err
			}
			line.Color = curve.Color
			p.Add(line)
		}

		points.Shape = draw.CircleGlyph{}
		points.Color = curve.Color

		p.Add(points)
	}

	log.Printf("before saving image to %s", filePath)
	dir := path.Dir(filePath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	return p.Save(30*vg.Centimeter, 15*vg.Centimeter, filePath)
}

func drawDynamicGraph(filepath string, data PlotData) error {
	fig := &grob.Fig{
		Data: grob.Traces{
			&grob.Scatter{
				X:     data.Curves[0].X,
				Y:     data.Curves[0].Y,
				Mode:  "markers",
				Xaxis: data.XAxisLabel,
				Yaxis: data.YAxisLabel,
			},
			&grob.Scatter{
				X:     data.Curves[1].X,
				Y:     data.Curves[1].Y,
				Mode:  "lines",
				Xaxis: data.XAxisLabel,
				Yaxis: data.YAxisLabel,
			},
		},
		Layout: &grob.Layout{
			Title: &grob.LayoutTitle{
				Text: data.Title,
			},
			Xaxis: &grob.LayoutXaxis{Type: grob.LayoutXaxisTypeDate},
		},
	}

	//offline.Show(fig)
	offline.ToHtml(fig, filepath)

	return nil
}

func transformForStaticGraph(x []string, y []float64) (plotter.XYs, error) {
	pts := make(plotter.XYs, len(x))

	for cont := 0; cont < len(x); cont++ {
		parsed, err := time.Parse(DateFormat, x[cont])
		if err != nil {
			return nil, err
		}
		date := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC).Unix()
		pts[cont].X = float64(date)
		pts[cont].Y = y[cont]

	}

	return pts, nil
}

func plotWeeklyGraph(opts weeklyGraphOpts) error {

	var errs []error
	metrics := strings.Split(opts.metricList, ",")
	for _, metric := range metrics {
		data, err := gatherPlotData(opts.weeklyReportsDir, opts.resource, ResultType(metric))
		if err != nil {
			fmt.Println("error gathering data for metric", err)
			fmt.Println("ignoring")
			continue
		}
		if !opts.plotlyHTML {
			err = drawStaticGraph(filepath.Join(opts.weeklyReportsDir, opts.resource, metric, "plot.png"), PlotData{
				Title:      fmt.Sprintf("Weekly %s for %s", metric, opts.resource),
				XAxisLabel: "Start date of week",
				YAxisLabel: "Metric Value",
				Curves:     data,
			})
			if err != nil {
				errs = append(errs, err)
				fmt.Println("error drawing a static graph for metric", err)
				fmt.Println("ignoring")
			}
		}
		// todo: plotly HTML
		err = drawDynamicGraph(filepath.Join(opts.weeklyReportsDir, opts.resource, metric, "index.html"), PlotData{
			Title:      fmt.Sprintf("Weekly %s for %s", metric, opts.resource),
			XAxisLabel: "Start date of week",
			YAxisLabel: "Metric Value",
			Curves:     data,
		})
		if err != nil {
			errs = append(errs, err)
			fmt.Println("error drawing a static graph for metric", err)
			fmt.Println("ignoring")
		}
	}
	return errors.NewAggregate(errs)
}
