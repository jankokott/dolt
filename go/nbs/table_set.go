// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/attic-labs/noms/go/d"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func newS3TableSet(s3 s3svc, bucket string) tableSet {
	return tableSet{p: s3TablePersister{s3, bucket}}
}

func newFSTableSet(dir string) tableSet {
	return tableSet{p: fsTablePersister{dir}}
}

// tableSet is an immutable set of persistable chunkSources.
type tableSet struct {
	chunkSources
	p tablePersister
}

// Prepend adds a memTable to an existing tableSet, compacting |mt| and
// returning a new tableSet with newly compacted table added.
func (ts tableSet) Prepend(mt *memTable) tableSet {
	if tableHash, chunkCount := ts.p.Compact(mt, ts); chunkCount > 0 {
		newTables := make(chunkSources, len(ts.chunkSources)+1)
		newTables[0] = ts.p.Open(tableHash, chunkCount)
		copy(newTables[1:], ts.chunkSources)
		return tableSet{newTables, ts.p}
	}
	return ts
}

// Union returns a new tableSet holding the union of the tables managed by
// |ts| and those specified by |specs|.
func (ts tableSet) Union(specs []tableSpec) tableSet {
	newTables := make(chunkSources, len(ts.chunkSources))
	known := map[addr]struct{}{}
	for i, t := range ts.chunkSources {
		known[t.hash()] = struct{}{}
		newTables[i] = ts.chunkSources[i]
	}

	for _, t := range specs {
		if _, present := known[t.name]; !present {
			newTables = append(newTables, ts.p.Open(t.name, t.chunkCount))
		}
	}
	return tableSet{newTables, ts.p}
}

func (ts tableSet) ToSpecs() []tableSpec {
	tableSpecs := make([]tableSpec, len(ts.chunkSources))
	for i, src := range ts.chunkSources {
		tableSpecs[i] = tableSpec{src.hash(), src.count()}
	}
	return tableSpecs
}

func (ts tableSet) Close() (err error) {
	for _, t := range ts.chunkSources {
		err = t.close() // TODO: somehow coalesce these errors??
	}
	return
}

type tablePersister interface {
	Compact(mt *memTable, haver chunkReader) (name addr, chunkCount uint32)
	Open(name addr, chunkCount uint32) chunkSource
}

type s3TablePersister struct {
	s3     s3svc
	bucket string
}

func (s3p s3TablePersister) Compact(mt *memTable, haver chunkReader) (name addr, chunkCount uint32) {
	name, data, chunkCount := mt.write(haver)

	if chunkCount > 0 {
		_, err := s3p.s3.PutObject(&s3.PutObjectInput{
			Bucket:        aws.String(s3p.bucket),
			Key:           aws.String(name.String()),
			Body:          bytes.NewReader(data),
			ContentLength: aws.Int64(int64(len(data))),
		})
		d.PanicIfError(err)
	}
	return name, chunkCount
}

func (s3p s3TablePersister) Open(name addr, chunkCount uint32) chunkSource {
	return newS3TableReader(s3p.s3, s3p.bucket, name, chunkCount)
}

type fsTablePersister struct {
	dir string
}

func (ftp fsTablePersister) Compact(mt *memTable, haver chunkReader) (name addr, chunkCount uint32) {
	tempName, name, chunkCount := func() (string, addr, uint32) {
		temp, err := ioutil.TempFile(ftp.dir, "nbs_table_")
		d.PanicIfError(err)
		defer checkClose(temp)

		name, data, chunkCount := mt.write(haver)
		io.Copy(temp, bytes.NewReader(data))
		return temp.Name(), name, chunkCount
	}()
	if chunkCount > 0 {
		err := os.Rename(tempName, filepath.Join(ftp.dir, name.String()))
		d.PanicIfError(err)
	} else {
		os.Remove(tempName)
	}
	return name, chunkCount
}

func (ftp fsTablePersister) Open(name addr, chunkCount uint32) chunkSource {
	return newMmapTableReader(ftp.dir, name, chunkCount)
}
