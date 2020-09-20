package docstore

import (
	"fmt"
	"io"
	"regexp"
	"time"
)

var (
	validDocRegex = regexp.MustCompile(`^[\.a-z0-9_-]*$`)
)

type Doc struct {
	Id             string
	LatestRevision int
}

type DocPage struct {
	Docs      []Doc
	NextToken string
	More      bool // True if there are more docs
}

type RevisionMetadata struct {
	DocId     string
	Id        int
	Timestamp time.Time
}

type Revision interface {
	Metadata() RevisionMetadata
	Read(p []byte) (n int, err error)
}

type RevisionPage struct {
	Revisions []RevisionMetadata
	NextToken string
	More      bool // True if there are more revisions
}

type DocStore interface {
	GetDoc(docId string) (rev Revision, err error)                      // Get the latest revision of a document
	GetRevision(docId string, revisionId int) (rev Revision, err error) // Get a specific revision of a document
	PutRevision(docId string, body io.Reader) (rev Revision, err error) // Put a new revision of a document. It will make a new doc if the DocId doesn't exist.
	ListDocs(token string) (page DocPage, err error)                    // List all the docs
	ListRevisions(docId string, token string) (RevisionPage, error)     // List all the revisions for a doc
}

// ValidateDocId returns an error if the docId doesn't match the validDocRegex
func ValidateDocId(docId string) error {
	if validDocRegex.Match([]byte(docId)) {
		return nil
	}

	return fmt.Errorf("Illegal characters in docId")
}
