// ingestdata.go

package deep6

//
// strucuture used by the ingest pipeeline to
// pass original data and derived data between
// each stage.
//
// Clssification of the data, and derivation of
// the object unique id etc. are governed by the
// configuration found in the
// ./config/datatype.toml file, see comments there
// for more details.
//
type IngestData struct {
	// Unique id for the object being processed
	// will be derived from the inbound json object
	// or created by the pipeline
	N3id string
	// The declared type of the objecct
	// such as a SIF StudentPersonal
	// for data with no type system, will use
	// the object datamodel, all xAPI objects
	// for instance end up as type XAPI
	Type string
	// The datamodel that the object being processed
	// appears to belong to based on the
	// datatype.toml classification
	// if no model can be derived will default to JSON
	DataModel string
	// The unmarshaled json of the object
	// as a map[string]interface{}
	RawData map[string]interface{}
	// The specifications for which features of an object should
	// be surfaced as links within the graph.
	// Provided in ./config/datatype.toml.
	// Searches triple predicates for the spec, so use e.g.
	// ".RefId" to find the precise refid of a SIF object
	// but use "RefId" (no dot) to find refids of referenced objects
	// such as SchoolInfoRefId
	LinkSpecs []string
	// Array of  values extracted during classification
	// which will be concatenated to make
	// a unique property identifier for objects
	// that have no discriminating features, e.g.
	// a syllabus has a stage, but so do lessons
	// it has a subject, but so do lessons and subjects
	// so to avoid filtering in a traversal
	// a combination of stage and subject will link
	// to only one sylabus.
	UniqueValues []string
	//
	// The resulting psuedo-unique key for this object.
	//
	Unique string
	// Potential links are derived from the inbound object
	// but need to be verified and written, this is done
	// by different parts of the ingest pipeline and so
	// are carried between stages in this slice
	LinkCandidates []Triple
	// The slice of hexastore triples parsed
	// from the original inbound data object
	Triples []Triple
	// The set of generated triiples that link
	// the features requested in LinkSpecs to the
	// rest of the graph
	LinkTriples []Triple
}
