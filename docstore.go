package docstore

import (
	"io"
	"time"
)

type Doc struct {
	Id, LatestRevision string
}

type DocPage struct {
	Docs []Doc
	NextToken string
	More bool // True if there are more docs
}

type RevisionMetadata struct {
	DocId, Id, PreviousRevision string
	Timestamp                   time.Time
}

type Revision interface {
	Metadata() RevisionMetadata
	Read(p []byte) (n int, err error)
}

type RevisionPage struct {
	Revisions []Revision
	NextToken string
	More bool // True if there are more revisions
}

type DocStore interface {
	GetDoc(docId string) (rev Revision, err error) // Get the latest revision of a document
	GetRevision(docId, revisionId string) (rev Revision, err error) // Get a specific revision of a document
	PutRevision(docId string, body io.Reader) (rev Revision, err error) // Put a new revision of a document. It will make a new doc if the DocId doesn't exist.
	ListDocs(token string) (page DocPage, err error) // List all the docs
	// ListRevisions(docId string, token string) (RevisionPage, error) // List all the revisions for a doc
}

func GenerateRevisionId() (time.Time, string) {
	now := time.Now()
	return now, now.Format(time.RFC3339)
}
