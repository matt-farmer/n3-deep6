// objectclassifier.go

package deep6

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nats-io/nuid"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

//
// Identifies & classifies the object passed in from the
// upstream reader.
//
// Uses the config in ./config/datatype.toml for deriving the
// data model, unique id etc.
//
// ctx - context to manage the pipeline
// in - channel providing map[string]interface{} containing
// the json data
//
func objectClassifier(ctx context.Context, in <-chan map[string]interface{}) (
	<-chan IngestData, // emits IngestData objects with classification elements
	<-chan error, // emits errors encountered to the pipeline manager
	error) { // any error encountered when creating this component

	out := make(chan IngestData)
	errc := make(chan error, 1)

	// load the classifier definitions;
	// each data-model type characterised by properties of the
	// json data.
	//
	type classifier struct {
		Data_model     string
		Required_paths []string
		N3id           string
		Links          []string
		Unique         []string
	}
	type classifiers struct {
		Classifier []classifier
	}
	var c classifiers
	if _, err := toml.DecodeFile("./config/datatypes.toml", &c); err != nil {
		return nil, nil, err
	}

	go func() {
		defer close(out)
		defer close(errc)
		for jsonMap := range in { // read json object (map) from upstream source

			rawJson, err := json.Marshal(jsonMap) // we need json bytes for use with gjson
			if err != nil {
				errc <- errors.Wrap(err, "json marshal error")
				return
			}

			igd := IngestData{}
			classified := false
			var dataModel, objectType, n3id, unique string
			var links, uniqueVals []string
			//
			// check the data by comparing with the known
			// classificaiton attributes from the config
			//
			for _, classifier := range c.Classifier {
				// extract the fields required for a synthetic unique id
				// if specified
				if len(classifier.Unique) > 0 {
					results := gjson.GetManyBytes(rawJson, classifier.Unique...)
					uniqueVals = make([]string, 0)
					for _, r := range results {
						if r.Exists() {
							uniqueVals = append(uniqueVals, r.String())
						}
					}
					unique = strings.Join(uniqueVals, "-")
				}
				// now apply classification
				results := gjson.GetManyBytes(rawJson, classifier.Required_paths...)
				found := 0
				for _, r := range results {
					if r.Exists() {
						found++
					}
				}
				if len(classifier.Required_paths) == found {
					classified = true
				}
				if classified {
					// find the unique identifier for this object
					// if no id available use a nuid
					result := gjson.GetBytes(rawJson, classifier.N3id)
					if result.Exists() {
						n3id = result.String()
					} else {
						n3id = nuid.Next()
					}
					dataModel = classifier.Data_model
					// collect link fields for this data type
					links = classifier.Links
					break
				}
			}

			// default if model isn't classified
			if !classified {
				dataModel = "JSON"
			}

			// set the object type
			// if only 1 top level key, derive object type from it (SIF)
			// otherwise default to the datamodel as type (eg. xAPI)
			keys := []string{}
			for k := range jsonMap {
				keys = append(keys, k)
			}
			if len(keys) == 1 {
				objectType = keys[0]
			} else {
				objectType = dataModel

			}

			//
			// store metadata back into the map itself
			//
			jsonMap["is-a"] = objectType
			if len(unique) > 0 {
				jsonMap["unique"] = unique
				igd.Unique = unique
			}
			igd.DataModel = dataModel
			igd.Type = objectType
			igd.N3id = n3id
			igd.LinkSpecs = links
			igd.RawData = jsonMap
			igd.UniqueValues = uniqueVals

			select {
			case out <- igd: // pass the data package on to the next stage
			case <-ctx.Done(): // listen for pipeline shutdown
				return
			}

		}
	}()

	return out, errc, nil

}
