package flakefinder

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"path"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
)

const (
	BucketName       = "kubevirt-prow"
	ReportsPath      = "reports/flakefinder"
	ReportFilePrefix = "flakefinder-"
	PreviewPath      = "preview"
)

//listGcsObjects get the slice of gcs objects under a given path
func ListGcsObjects(ctx context.Context, client *storage.Client, bucketName, prefix, delim string) (
	[]string, error) {

	var objects []string
	it := client.Bucket(bucketName).Objects(ctx, &storage.Query{
		Prefix:    prefix,
		Delimiter: delim,
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return objects, fmt.Errorf("error iterating: %v", err)
		}

		if attrs.Prefix != "" {
			objects = append(objects, path.Base(attrs.Prefix))
		}
	}
	logrus.Info("end of listGcsObjects(...)")
	return objects, nil
}

func ReadGcsObjectAttrs(ctx context.Context, client *storage.Client, bucket, object string) (attrs *storage.ObjectAttrs, err error) {
	logrus.Infof("Trying to read gcs object attrs '%s' in bucket '%s'\n", object, bucket)
	attrs, err = client.Bucket(bucket).Object(object).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("Cannot read attrs from %s in bucket '%s'", object, bucket)
	}
	return
}

func WriteTemplateToOutput(tpl string, parameters interface{}, writer io.Writer) error {
	t, err := template.New("report").Parse(tpl)
	if err != nil {
		return fmt.Errorf("failed to load template: %v", err)
	}

	err = t.Execute(writer, parameters)
	return err
}

func CreateOutputWriter(client *storage.Client, ctx context.Context, outputPath string) io.WriteCloser {
	reportIndexObject := client.Bucket(BucketName).Object(path.Join(outputPath, "index.html"))
	log.Printf("Report index page will be written to gs://%s/%s", BucketName, reportIndexObject.ObjectName())
	reportIndexObjectWriter := reportIndexObject.NewWriter(ctx)
	return reportIndexObjectWriter
}

