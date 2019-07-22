// deep6.go

package deep6

import (
	"log"
	"os"

	"github.com/dgraph-io/badger"
	boom "github.com/tylertreat/BoomFilters"
)

type Deep6DB struct {
	//
	// the underlying badger k/v store used by D6
	//
	db *badger.DB
	//
	// manages parallel async writing to db
	//
	iwb *badger.WriteBatch
	//
	// another 'writer' used for deletes
	//
	rwb *badger.WriteBatch
	//
	// sbf used to record links
	//
	sbf *boom.ScalableBloomFilter
	//
	// set level of audit ouput, one of: none, basic, high
	//
	AuditLevel string
}

//
// Open the database using the specified directory
// It will be created if it doesn't exist.
//
func OpenFromFile(folderPath string) (*Deep6DB, error) {

	err := os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	options := badger.DefaultOptions(folderPath)
	// options = options.WithSyncWrites(false) // speed optimisation if required
	db, err := badger.Open(options)
	if err != nil {
		return nil, err
	}
	iwb := db.NewWriteBatch()
	rwb := db.NewWriteBatch()

	sbf := openSBF()

	return &Deep6DB{db: db, iwb: iwb, rwb: rwb, sbf: sbf, AuditLevel: "high"}, nil
}

//
// Open a d6db will use the local path
// ./db/badger by default
//
func Open() (*Deep6DB, error) {

	// if no filename provided will create locally
	return OpenFromFile("./db/d6")

}

//
// shuts down the and ensures all writes are
// committed.
//
func (d6 *Deep6DB) Close() {
	log.Println("closing database...")
	err := d6.iwb.Flush()
	if err != nil {
		log.Println("error flushing ingest writebatch: ", err)
	}
	err = d6.rwb.Flush()
	if err != nil {
		log.Println("error flushing delete writebatch: ", err)
	}
	err = d6.db.Close()
	if err != nil {
		log.Println("error closing datastore:", err)
	}
	log.Println("...database closed")
	log.Println("saving sbf....")
	saveSBF(d6.sbf)
	log.Println("...sbf saved.")

}
