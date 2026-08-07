//go:build !s3

package main

// makeS3Store only exists when built with the "s3" tag (STORAGE=s3). Without
// the tag we report that S3 support must be enabled at build time.
func makeS3Store() (store, error) {
	return nil, &noS3Error{}
}

type noS3Error struct{}

func (e *noS3Error) Error() string {
	return "STORAGE=s3 requires building with: go build -tags s3"
}