// Copyright 2017 Quentin Machu & eco authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package s3

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	ss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"

	"github.com/quentin-m/etcd-cloud-operator/pkg/providers"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
)

func init() {
	snapshot.Register("s3", &s3{})
}

type s3 struct {
	config config
	region string
}

type config struct {
	Bucket string `yaml:"bucket"`
}

func (s *s3) Configure(providerConfig snapshot.Config) error {
	if err := providers.ParseParams(providerConfig.Params, &s.config); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	if s.config.Bucket == "" {
		return errors.New("invalid configuration: bucket name is missing")
	}

	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create aws session: %v", err)
	}

	ec2meta := ec2metadata.New(sess)
	if !ec2meta.Available() {
		return errors.New("application is not running on aws ec2")
	}

	s.region, err = ec2meta.Region()
	if err != nil {
		return fmt.Errorf("failed to retrieve aws ec2 region: %v", err)
	}

	if _, err := s.Info(); err != nil && err != snapshot.ErrNoSnapshot {
		return fmt.Errorf("failed to validate aws s3 configuration: %v", err)
	}

	return nil
}

func (s *s3) Save(r io.ReadCloser, metadata *snapshot.Metadata) error {
	key := metadata.Filename()

	sess, err := session.NewSession(aws.NewConfig().WithRegion(s.region))
	if err != nil {
		return fmt.Errorf("failed to create aws session: %v", err)
	}
	s3s := ss3.New(sess)

	_, err = s3manager.NewUploader(sess).Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	if err != nil {
		s3s.DeleteObject(&ss3.DeleteObjectInput{})
		return fmt.Errorf("failed to upload aws s3 object: %v", err)
	}

	resp, err := s3s.HeadObject(&ss3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.WithError(err).Warnf("failed to get aws s3 object size for %q")
		return nil
	}

	metadata.Size = *resp.ContentLength
	return nil
}

func (s *s3) Get(metadata *snapshot.Metadata) (string, bool, error) {
	sess, err := session.NewSession(aws.NewConfig().WithRegion(s.region))
	if err != nil {
		return "", true, fmt.Errorf("failed to create aws session: %v", err)
	}

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", true, err
	}

	if _, err := s3manager.NewDownloader(sess).Download(f, &ss3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(metadata.Name),
	}); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", true, fmt.Errorf("failed to get aws s3 object: %v", err)
	}

	f.Sync()
	f.Close()

	return f.Name(), true, nil
}

func (s *s3) Info() (*snapshot.Metadata, error) {
	sess, err := session.NewSession(aws.NewConfig().WithRegion(s.region))
	if err != nil {
		return nil, fmt.Errorf("failed to create aws session: %v", err)
	}
	s3s := ss3.New(sess)

	resp, err := s3s.ListObjects(&ss3.ListObjectsInput{Bucket: aws.String(s.config.Bucket)})
	if err != nil {
		return nil, fmt.Errorf("failed to list aws s3 objects: %v", err)
	}

	var metadatas []*snapshot.Metadata
	for _, obj := range resp.Contents {
		metadata, err := snapshot.NewMetadata(*obj.Key, -1, *obj.Size, s)
		if err != nil {
			log.Warnf("failed to parse metadata for snapshot %v", *obj.Key)
			continue
		}
		metadatas = append(metadatas, metadata)
	}
	if len(metadatas) == 0 {
		return nil, snapshot.ErrNoSnapshot
	}
	sort.Sort(snapshot.MetadataSorter(metadatas))

	return metadatas[len(metadatas)-1], nil
}

func (s *s3) Purge(ttl time.Duration) error {
	sess, err := session.NewSession(aws.NewConfig().WithRegion(s.region))
	if err != nil {
		return fmt.Errorf("failed to create aws session: %v", err)
	}
	s3s := ss3.New(sess)

	resp, err := s3s.ListObjects(&ss3.ListObjectsInput{Bucket: aws.String(s.config.Bucket)})
	if err != nil {
		return fmt.Errorf("failed to list aws s3 objects: %v", err)
	}

	for _, item := range resp.Contents {
		if time.Since(*item.LastModified) > ttl {
			log.Infof("purging snapshot file %q because it is that older than %v", *item.Key, ttl)

			_, err := s3s.DeleteObject(&ss3.DeleteObjectInput{
				Bucket: aws.String(s.config.Bucket),
				Key:    item.Key,
			})
			if err != nil {
				log.WithError(err).Warn("failed to remove aws s3 object")
			}
		}
	}

	return nil
}
