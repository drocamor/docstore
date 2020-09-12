package awsdocstore

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/drocamor/docstore"
	"io"
	"io/ioutil"
	"time"
)

type AwsDocStore struct {
	ddb                     *dynamodb.DynamoDB
	docTable, revisionTable string
}

type AwsDocStoreOption func(*AwsDocStore)

type AwsRevision struct {
	DocId     string
	Id        int
	Timestamp time.Time
	Body      []byte
	reader    *bytes.Reader
}

func (r *AwsRevision) Metadata() docstore.RevisionMetadata {
	return docstore.RevisionMetadata{
		DocId:     r.DocId,
		Id:        r.Id,
		Timestamp: r.Timestamp,
	}
}

func (r *AwsRevision) Read(p []byte) (n int, err error) {
	if r.reader == nil {
		// TODO get the body
		err = fmt.Errorf("uninitialized reader")
		return
	}

	return r.reader.Read(p)
}

func WithDocTable(s string) AwsDocStoreOption {
	return func(d *AwsDocStore) {
		d.docTable = s
	}
}

func WithRevisionTable(s string) AwsDocStoreOption {
	return func(d *AwsDocStore) {
		d.revisionTable = s
	}
}

var defaultAwsDocStore = AwsDocStore{
	docTable:      "docs",
	revisionTable: "revisions",
}

func New(opts ...AwsDocStoreOption) *AwsDocStore {
	ds := defaultAwsDocStore

	for _, o := range opts {
		o(&ds)
	}

	ds.ddb = dynamodb.New(session.New())

	return &ds
}

func (ds *AwsDocStore) GetDoc(docId string) (rev docstore.Revision, err error) {
	// Get the doc record
	docKey := struct{ Id string }{Id: docId}
	docKeyAv, err := dynamodbattribute.MarshalMap(docKey)
	if err != nil {
		return
	}
	getDocReq := (&dynamodb.GetItemInput{}).
		SetTableName(ds.docTable).
		SetKey(docKeyAv)

	resp, err := ds.ddb.GetItem(getDocReq)
	if err != nil {
		return
	}

	if len(resp.Item) == 0 {
		err = fmt.Errorf("Doc not found.")
		return
	}

	// Unmarshal the response
	var doc docstore.Doc

	err = dynamodbattribute.UnmarshalMap(resp.Item, &doc)
	if err != nil {
		return
	}

	// Use GetRevision
	return ds.GetRevision(doc.Id, doc.LatestRevision)

}

func (ds *AwsDocStore) GetRevision(docId string, revisionId int) (rev docstore.Revision, err error) {
	// TODO - validate that the doc actually exists first

	// Make the key

	revKey := struct {
		Id    int
		DocId string
	}{
		Id:    revisionId,
		DocId: docId,
	}

	revKeyAv, err := dynamodbattribute.MarshalMap(revKey)
	if err != nil {
		return
	}

	// Get the revision
	getRevReq := (&dynamodb.GetItemInput{}).
		SetTableName(ds.revisionTable).
		SetKey(revKeyAv)

	resp, err := ds.ddb.GetItem(getRevReq)
	if err != nil {
		return
	}

	if len(resp.Item) == 0 {
		err = fmt.Errorf("Revision not found.")
		return
	}

	aRev := AwsRevision{}

	err = dynamodbattribute.UnmarshalMap(resp.Item, &aRev)
	if err != nil {
		return
	}
	aRev.reader = bytes.NewReader(aRev.Body)

	rev = &aRev
	return

}

func (ds *AwsDocStore) PutRevision(docId string, body io.Reader) (rev docstore.Revision, err error) {

	// Create a new revision ID
	timestamp := time.Now()

	// Increment the LatestRevision attribute.
	/*
		update := expression.Set(
			expression.Name("LatestRevision"),
			expression.Name("LatestRevision").Plus(expression.Value(1)),
		)
	*/
	update := expression.Add(
		expression.Name("LatestRevision"),
		expression.Value(1),
	)

	expr, err := expression.NewBuilder().
		WithUpdate(update).
		Build()

	if err != nil {
		return
	}

	// Create the key
	docKey := struct{ Id string }{Id: docId}
	docKeyAv, err := dynamodbattribute.MarshalMap(docKey)
	if err != nil {
		return
	}

	// Update the docs table to change the LatestRevision. The return value includes the revision ID for the new revision.
	updateDocReq := (&dynamodb.UpdateItemInput{}).
		SetTableName(ds.docTable).
		SetReturnValues("UPDATED_NEW").
		SetKey(docKeyAv)

	updateDocReq.ExpressionAttributeNames = expr.Names()
	updateDocReq.ExpressionAttributeValues = expr.Values()
	updateDocReq.UpdateExpression = expr.Update()

	resp, err := ds.ddb.UpdateItem(updateDocReq)

	if err != nil {
		return
	}

	// Determine the previous revision of the doc
	var revId int
	if len(resp.Attributes) > 0 {
		updatedValues := struct{ LatestRevision int }{}
		err = dynamodbattribute.UnmarshalMap(resp.Attributes, &updatedValues)
		if err != nil {
			return
		}

		revId = updatedValues.LatestRevision
	}

	// Create a new version of the doc

	b, err := ioutil.ReadAll(body)
	if err != nil {
		return
	}

	rev = &AwsRevision{
		DocId:     docId,
		Id:        revId,
		Timestamp: timestamp,
		Body:      b,
		reader:    bytes.NewReader(b),
	}

	revAv, err := dynamodbattribute.MarshalMap(rev)
	if err != nil {
		return
	}

	revInput := (&dynamodb.PutItemInput{}).
		SetTableName(ds.revisionTable).
		SetItem(revAv)

	_, err = ds.ddb.PutItem(revInput)

	return
}

func (ds *AwsDocStore) ListDocs(token string) (page docstore.DocPage, err error) {
	scanInput := (&dynamodb.ScanInput{}).
		SetTableName(ds.docTable)

	/*
		// If there is a token it will be the last evaluated key
		if token != "" {

		}
	*/
	resp, err := ds.ddb.Scan(scanInput)
	if err != nil {
		return
	}

	var docs []docstore.Doc

	err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &docs)
	if err != nil {
		return
	}

	page.Docs = docs
	return

}
