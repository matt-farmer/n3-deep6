package deep6

import (
	"context"
	"io"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	boom "github.com/tylertreat/BoomFilters"
)

//
// Runs the ingest pipleine to consume json data.
//
// Needs db and wb as these are presumed to be in use
// by other pipelines or application features.
//
// eg. db/wb are used here for loading data
// but db can also be in use to support queries in parallel.
//
// db - instance of a badger db
// wb - badger.WriteBatch, a fast write manager provided by the db
// sbf - bloom filter used to capture required graph links as data traverses the pipeline
// r - the io.Reader (file, http body etc.) to be ingested
// auditLevel - one of: none, basic, high
//
func runIngestWithReader(db *badger.DB, wb *badger.WriteBatch, sbf *boom.ScalableBloomFilter, r io.Reader, auditLevel string) error {

	// set up a context to manage ingest pipeline
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// monitor all error channels
	var errcList []<-chan error

	//
	// build the pipleine by connecting all stages
	//
	jsonOut, errc, err := jsonReaderSource(ctx, r)
	if err != nil {
		return errors.Wrap(err, "Error: cannot create json-reader source component: ")
	}
	errcList = append(errcList, errc)

	classOut, errc, err := objectClassifier(ctx, jsonOut)
	if err != nil {
		return errors.Wrap(err, "Error: cannot create object-classifier component: ")
	}
	errcList = append(errcList, errc)

	genOut, errc, err := tupleGenerator(ctx, classOut)
	if err != nil {
		return errors.Wrap(err, "Error: cannot create tuple-generator component: ")
	}
	errcList = append(errcList, errc)

	writerOut, errc, err := tripleWriter(ctx, wb, genOut)
	if err != nil {
		return errors.Wrap(err, "Error: cannot create triple-writer component: ")
	}
	errcList = append(errcList, errc)

	linkerOut, errc, err := linkParser(ctx, sbf, writerOut)
	if err != nil {
		return errors.Wrap(err, "Error: cannot create link-parser component: ")
	}
	errcList = append(errcList, errc)

	builderOut, errc, err := linkBuilder(ctx, db, wb, linkerOut)
	if err != nil {
		return errors.Wrap(err, "Error: cannot create link-builder component: ")
	}
	errcList = append(errcList, errc)

	lwriterOut, errc, err := linkWriter(ctx, wb, builderOut)
	if err != nil {
		return errors.Wrap(err, "Error: cannot create link-writer component: ")
	}
	errcList = append(errcList, errc)

	errc, err = ingestAuditSink(ctx, auditLevel, lwriterOut)
	if err != nil {
		return errors.Wrap(err, "Error: cannot create audit-sink component: ")
	}
	errcList = append(errcList, errc)

	// monitor progress
	err = WaitForPipeline(errcList...)

	return err

}
