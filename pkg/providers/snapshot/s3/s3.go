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
	"time"
	
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	ss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
	log "github.com/sirupsen/logrus"
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
	
	if _, _, err := s.latestSnapshotKey(); err != nil && err != snapshot.ErrNoSnapshot {
		return fmt.Errorf("failed to validate aws s3 configuration: %v", err)
	}
	
	return nil
}

func (s *s3) Save(r io.ReadCloser, name string, rev int64) (int64, error) {
	key := snapshot.Name(rev, name)

	sess, err := session.NewSession(aws.NewConfig().WithRegion(s.region))
	if err != nil {
		return 0, fmt.Errorf("failed to create aws session: %v", err)
	}
	s3s := ss3.New(sess)

	_, err = s3manager.NewUploader(sess).Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to upload aws s3 object: %v", err)
	}

	resp, err := s3s.HeadObject(&ss3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.WithError(err).Warnf("failed to get aws s3 object size for %q")
		return 0, nil
	}

	return *resp.ContentLength, err
}

func (s *s3) Latest() (io.ReadCloser, int64, int64, error) {
	key, rev, err := s.latestSnapshotKey()
	if err != nil {
		return nil, 0, 0, err
	}

	sess, err := session.NewSession(aws.NewConfig().WithRegion(s.region))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create aws session: %v", err)
	}
	s3s := ss3.New(sess)

	resp, err := s3s.GetObject(&ss3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get aws s3 object: %v", err)
	}

	return resp.Body, *resp.ContentLength, rev, nil
}

func (s *s3) LatestRev() (int64, error) {
	_, rev, err := s.latestSnapshotKey()
	return rev, err
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

func (s *s3) latestSnapshotKey() (string, int64, error) {
	sess, err := session.NewSession(aws.NewConfig().WithRegion(s.region))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create aws session: %v", err)
	}
	s3s := ss3.New(sess)

	resp, err := s3s.ListObjects(&ss3.ListObjectsInput{Bucket: aws.String(s.config.Bucket)})
	if err != nil {
		return "", 0, fmt.Errorf("failed to list aws s3 objects: %v", err)
	}

	var names []string
	for _, obj := range resp.Contents {
		names = append(names, *obj.Key)
	}

	return snapshot.LatestFromNames(names)
}
