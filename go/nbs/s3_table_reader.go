// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"fmt"
	"io"

	"github.com/attic-labs/noms/go/d"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

const s3RangePrefix = "bytes"

type s3TableReader struct {
	tableReader
	s3     s3svc
	bucket string
	h      addr
}

type s3svc interface {
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

func newS3TableReader(s3 s3svc, bucket string, h addr, chunkCount uint32) chunkSource {
	source := &s3TableReader{s3: s3, bucket: bucket, h: h}

	size := indexSize(chunkCount) + footerSize
	buff := make([]byte, size)

	n, err := source.readRange(buff, fmt.Sprintf("%s=-%d", s3RangePrefix, size))
	d.PanicIfError(err)
	d.PanicIfFalse(size == uint64(n))

	source.tableReader = newTableReader(buff, source)
	d.PanicIfFalse(chunkCount == source.count())
	return source
}

func (s3tr *s3TableReader) close() error {
	return nil
}

func (s3tr *s3TableReader) hash() addr {
	return s3tr.h
}

func (s3tr *s3TableReader) ReadAt(p []byte, off int64) (n int, err error) {
	end := off + int64(len(p)) - 1 // insanely, the HTTP range header specifies ranges inclusively.
	rangeHeader := fmt.Sprintf("%s=%d-%d", s3RangePrefix, off, end)
	return s3tr.readRange(p, rangeHeader)
}

func (s3tr *s3TableReader) readRange(p []byte, rangeHeader string) (n int, err error) {
	result, err := s3tr.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s3tr.bucket),
		Key:    aws.String(s3tr.hash().String()),
		Range:  aws.String(rangeHeader),
	})
	d.PanicIfError(err)
	d.PanicIfFalse(*result.ContentLength == int64(len(p)))

	return io.ReadFull(result.Body, p)
}
