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

package tester

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"go.etcd.io/etcd/client/v3"
	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
)

const (
	stresserKeySize           int = 100
	stresserKeySizeBig        int = 32*1024 + 1
	stresserKeySuffixRange    int = 250000
	stresserKeyTxnOps         int = 32
	stresserKeyTxnSuffixRange int = 100
	stresserThreads           int = 16
	stresserQPS               int = 1000
)

type stresser struct {
	ctx    context.Context
	cancel func()

	client      *clientv3.Client
	rateLimiter *rate.Limiter

	threads int
	wg      sync.WaitGroup
}

func newStresser() *stresser {
	ctx, cancel := context.WithCancel(context.Background())

	return &stresser{
		ctx:    ctx,
		cancel: cancel,

		threads:     stresserThreads,
		rateLimiter: rate.NewLimiter(rate.Limit(stresserQPS), stresserQPS),
	}
}

func (s *stresser) Start(c *clientv3.Client) {
	s.client = c
	for i := 0; i < s.threads; i++ {
		s.wg.Add(1)
		go s.run()
	}
}

func (s *stresser) Stop() {
	s.cancel()
	s.wg.Wait()
}

func (s *stresser) run() {
	defer s.wg.Done()

	for {
		if err := s.rateLimiter.Wait(s.ctx); err == context.Canceled {
			return
		}

		sCtx, sCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		err, _ := s.choose()(sCtx)
		sCancel()

		promRequestsTotal.Inc()
		if err != nil {
			promFailedRequestsTotal.Inc()
		}
	}
}

func (s *stresser) choose() func(context.Context) (error, int64) {
	r := rand.Float32()
	switch {
	case r < 0.15:
		return s.put
	case r < 0.30:
		return s.putTxn
	case r < 0.45:
		return s.putBig
	case r < 0.60:
		return s.getSingle
	case r < 0.75:
		return s.getRange
	case r < 0.90:
		return s.deleteSingle
	case r < 1.00:
		return s.deleteRange
	default:
		return nil
	}
}

func (s *stresser) put(ctx context.Context) (error, int64) {
	_, err := s.client.Put(
		ctx,
		fmt.Sprintf("foo%016x", rand.Intn(stresserKeySuffixRange)),
		string(randBytes(stresserKeySize)),
	)
	return err, 1
}

func (s *stresser) putTxn(ctx context.Context) (error, int64) {
	keys := make([]string, stresserKeyTxnSuffixRange)
	for i := range keys {
		keys[i] = fmt.Sprintf("/k%03d", i)
	}

	ks := make(map[string]struct{}, stresserKeyTxnOps)
	for len(ks) != stresserKeyTxnOps {
		ks[keys[rand.Intn(len(keys))]] = struct{}{}
	}
	selected := make([]string, 0, stresserKeyTxnOps)
	for k := range ks {
		selected = append(selected, k)
	}

	// Base transaction.
	com, delOp, putOp := getTxnReqs(selected[0], "bar00")
	cmps := []clientv3.Cmp{com}
	succs := []clientv3.Op{delOp}
	fails := []clientv3.Op{putOp}

	// Nest additional transactions in the base transaction's success and failure cases.
	for i := 1; i < stresserKeyTxnOps; i++ {
		k, v := selected[i], fmt.Sprintf("bar%02d", i)

		com, delOp, putOp = getTxnReqs(k, v)
		txn := clientv3.OpTxn([]clientv3.Cmp{com}, []clientv3.Op{delOp}, []clientv3.Op{putOp})

		succs = append(succs, txn)
		succs = append(fails, txn)
	}
	_, err := s.client.Do(ctx, clientv3.OpTxn(cmps, succs, fails))
	return err, int64(stresserKeyTxnOps)
}

func (s *stresser) putBig(ctx context.Context) (error, int64) {
	_, err := s.client.Put(
		ctx,
		fmt.Sprintf("foo%016x", rand.Intn(stresserKeySuffixRange)),
		string(randBytes(stresserKeySizeBig)),
	)
	return err, 1
}

func (s *stresser) getSingle(ctx context.Context) (error, int64) {
	_, err := s.client.Get(ctx, fmt.Sprintf("foo%016x", rand.Intn(stresserKeySuffixRange)))
	return err, 0
}

func (s *stresser) getRange(ctx context.Context) (error, int64) {
	start := rand.Intn(stresserKeySuffixRange)
	end := start + 500
	_, err := s.client.Get(ctx, fmt.Sprintf("foo%016x", start), clientv3.WithRange(fmt.Sprintf("foo%016x", end)))
	return err, 0
}

func (s *stresser) deleteSingle(ctx context.Context) (error, int64) {
	_, err := s.client.Delete(ctx, fmt.Sprintf("foo%016x", rand.Intn(stresserKeySuffixRange)))
	return err, 1
}

func (s *stresser) deleteRange(ctx context.Context) (error, int64) {
	start := rand.Intn(stresserKeySuffixRange)
	end := start + 500
	resp, err := s.client.Delete(ctx, fmt.Sprintf("foo%016x", start), clientv3.WithRange(fmt.Sprintf("foo%016x", end)))
	if err == nil {
		return nil, resp.Deleted
	}
	return err, 0
}

func getTxnReqs(key, val string) (com clientv3.Cmp, delOp clientv3.Op, putOp clientv3.Op) {
	// if key exists (version > 0)
	com = clientv3.Cmp{
		Key:         []byte(key),
		Target:      pb.Compare_VERSION,
		Result:      pb.Compare_GREATER,
		TargetUnion: &pb.Compare_Version{Version: 0},
	}
	delOp = clientv3.OpDelete(key)
	putOp = clientv3.OpPut(key, val)
	return com, delOp, putOp
}

func randBytes(size int) []byte {
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(int('a') + rand.Intn(26))
	}
	return data
}
