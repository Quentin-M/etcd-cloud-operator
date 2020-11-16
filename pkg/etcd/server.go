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

package etcd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	etcdcl "go.etcd.io/etcd/clientv3"
	etcdsnap "go.etcd.io/etcd/clientv3/snapshot"
	"go.uber.org/zap"

	"go.etcd.io/etcd/embed"
	"go.etcd.io/etcd/pkg/types"
	"google.golang.org/grpc/grpclog"

	"github.com/quentin-m/etcd-cloud-operator/pkg/logger"
	"github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot"
	_ "github.com/quentin-m/etcd-cloud-operator/pkg/providers/snapshot/etcd"
)

var ErrMemberRevisionTooOld = errors.New("member revision older than the minimum desired revision")

const (
	defaultStartTimeout          = 1800 * time.Second
	defaultStartRejoinTimeout    = 60 * time.Second
	defaultMemberCleanerInterval = 15 * time.Second
)

type Server struct {
	server    *embed.Etcd
	isRunning bool
	cfg       ServerConfig
}

type ServerConfig struct {
	Name                    string
	DataDir                 string
	DataQuota               int64
	PublicAddress           string
	PrivateAddress          string
	ClientSC                SecurityConfig
	PeerSC                  SecurityConfig
	UnhealthyMemberTTL      time.Duration
	AutoCompactionMode      string
	AutoCompactionRetention string

	// Optional, used in {Seed, Join} to periodically save snapshots.
	SnapshotProvider snapshot.Provider
	SnapshotInterval time.Duration
	SnapshotTTL      time.Duration

	// Internal, used in startServer.
	clusterState string
	initialPURLs map[string]string
}

func NewServer(cfg ServerConfig) *Server {
	return &Server{
		cfg: cfg,
	}
}

func (c *Server) Seed(snapshot *snapshot.Metadata) error {
	// Restore a snapshot if a provider is given.
	if snapshot != nil {
		if err := c.Restore(snapshot); err != nil {
			return fmt.Errorf("failed to restore snapshot: %v", err)
		}
	} else {
		// Remove the existing data directory.
		//
		// When there is a snapshot available, we let Restore take care of the data directory entirely.
		os.RemoveAll(c.cfg.DataDir)
	}

	// Set the internal configuration.
	c.cfg.clusterState = embed.ClusterStateFlagNew
	c.cfg.initialPURLs = map[string]string{c.cfg.Name: peerURL(c.cfg.PrivateAddress, c.cfg.PeerSC.TLSEnabled())}

	// Start the server.
	ctx, cancel := context.WithTimeout(context.Background(), defaultStartTimeout)
	defer cancel()
	return c.startServer(ctx)
}

func (c *Server) Join(cluster *Client) error {
	// List the existing members.
	ctx, cancel := context.WithTimeout(context.Background(), defaultStartTimeout)
	members, err := cluster.MemberList(ctx)
	defer cancel()

	if err != nil {
		return fmt.Errorf("failed to list cluster's members: %v", err)
	}

	// Set the internal configuration.
	c.cfg.initialPURLs = map[string]string{c.cfg.Name: peerURL(c.cfg.PrivateAddress, c.cfg.PeerSC.TLSEnabled())}
	for _, member := range members.Members {
		if member.Name == "" {
			continue
		}
		c.cfg.initialPURLs[member.Name] = member.PeerURLs[0]
	}
	c.cfg.clusterState = embed.ClusterStateFlagExisting

	// Check if we are listed as a member, and save the member ID if so.
	var memberID uint64
	for _, member := range members.Members {
		if c.cfg.Name == member.Name {
			memberID = member.ID
			break
		}
	}
	// Verify whether we have local data that would allow us to rejoin.
	_, localSnapErr := localSnapshotProvider(c.cfg.DataDir).Info()

	// Attempt to re-join the server directly if we are still a member, and we have local data.
	if memberID != 0 && localSnapErr == nil {
		log.Info("attempting to rejoin cluster under existing identity with local data")

		ctx, cancel := context.WithTimeout(context.Background(), defaultStartRejoinTimeout)
		defer cancel()
		if err := c.startServer(ctx); err == nil {
			return nil
		}

		log.Warn("failed to join as an existing member, resetting")
		if err := cluster.RemoveMember(c.cfg.Name, memberID); err != nil {
			log.WithError(err).Warning("failed to remove ourselves from the cluster's member list")
		}
	}
	os.RemoveAll(c.cfg.DataDir)

	// Add ourselves as a member.
	memberID, unlock, err := cluster.AddMember(c.cfg.Name, []string{peerURL(c.cfg.PrivateAddress, c.cfg.PeerSC.TLSEnabled())})
	if err != nil {
		return fmt.Errorf("failed to add ourselves as a member of the cluster: %v", err)
	}
	defer unlock()

	// Start the server.
	ctx, cancel = context.WithTimeout(context.Background(), defaultStartTimeout)
	defer cancel()
	if err := c.startServer(ctx); err != nil {
		cluster.RemoveMember(c.cfg.Name, memberID)
		return err
	}
	return nil
}

func (c *Server) Restore(metadata *snapshot.Metadata) error {
	log.Infof("restoring snapshot %q (rev: %016x, size: %.3f MB)", metadata.Name, metadata.Revision, toMB(metadata.Size))

	path, shouldDelete, err := metadata.Source.Get(metadata)
	if err != nil && err != snapshot.ErrNoSnapshot {
		return fmt.Errorf("failed to retrieve latest snapshot: %v", err)
	}
	if shouldDelete {
		defer os.Remove(path)
	}

	// Remove the existing data directory.
	//
	// We do it only after getting the snapshot, because in the case of the local 'etcd' snapshotter, the data is copied
	// directly from the data directory, to a temporary file when Get is called.
	os.RemoveAll(c.cfg.DataDir)

	restorePeerURL := peerURL(c.cfg.PrivateAddress, c.cfg.PeerSC.TLSEnabled())
	restoreCfg := etcdsnap.RestoreConfig{
		SnapshotPath:        path,
		Name:                c.cfg.Name,
		PeerURLs:            []string{restorePeerURL},
		InitialCluster:      fmt.Sprintf("%s=%s", c.cfg.Name, restorePeerURL),
		InitialClusterToken: embed.NewConfig().InitialClusterToken,
		OutputDataDir:       c.cfg.DataDir,
		SkipHashCheck:       true,
	}

	if err := etcdsnap.NewV3(zap.L()).Restore(restoreCfg); err != nil {
		return fmt.Errorf("etcdctl failed to restore:\n %s", err)
	}

	return nil
}

func (c *Server) Snapshot() error {
	t := time.Now()

	// Purge old snapshots in the background.
	go c.cfg.SnapshotProvider.Purge(c.cfg.SnapshotTTL)

	// Get the latest snapshotted revision.
	var minRev int64
	if metadata, err := c.cfg.SnapshotProvider.Info(); err == nil {
		minRev = metadata.Revision
	} else {
		if err != snapshot.ErrNoSnapshot {
			log.WithError(err).Warn("failed to find latest snapshot revision, snapshotting anyways")
		}
	}

	// Initiate a snapshot.
	rc, rev, err := c.snapshot(minRev)
	if err == ErrMemberRevisionTooOld {
		log.Infof("skipping snapshot: current revision %016x <= latest snapshot %016x", rev, minRev)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to initiate snapshot: %v", err)
	}
	defer rc.Close()

	// Save the incoming snapshot.
	metadata, _ := snapshot.NewMetadata(c.cfg.Name, rev, -1, c.cfg.SnapshotProvider)
	if err := c.cfg.SnapshotProvider.Save(rc, metadata); err != nil {
		return fmt.Errorf("failed to save snapshot: %v", err)
	}

	log.Infof("snapshot %q saved successfully in %v (%.2f MB)", metadata.Filename(), time.Since(t), toMB(metadata.Size))
	return nil
}

func (c *Server) SnapshotInfo() (*snapshot.Metadata, error) {
	var localSnap, cfgSnap *snapshot.Metadata
	var localErr, cfgErr error

	// Read snapshot info from the local etcd data, if etcd is not running (otherwise it'll get stuck).
	if !c.isRunning {
		localSnap, localErr = localSnapshotProvider(c.cfg.DataDir).Info()
		if localErr != nil && localErr != snapshot.ErrNoSnapshot {
			log.WithError(localErr).Warn("failed to retrieve local snapshot info")
		}
	}

	// Read snapshot info from the configured snapshot provider.
	cfgSnap, cfgErr = c.cfg.SnapshotProvider.Info()
	if cfgErr != nil && cfgErr != snapshot.ErrNoSnapshot {
		log.WithError(cfgErr).Warn("failed to retrieve snapshot info")
	}

	// Return the highest revision one, or the one that worked.
	if localErr == snapshot.ErrNoSnapshot && cfgErr == snapshot.ErrNoSnapshot {
		return nil, snapshot.ErrNoSnapshot
	}
	if localErr != nil && cfgErr != nil {
		return nil, errors.New("failed to retrieve snapshot info")
	}
	if cfgErr != nil || (localErr == nil && localSnap.Revision >= cfgSnap.Revision) {
		return localSnap, nil
	}
	return cfgSnap, cfgErr
}

func (c *Server) snapshot(minRevision int64) (io.ReadCloser, int64, error) {
	// Get the current revision and compare with the minimum requested revision.
	revision := c.server.Server.KV().Rev()
	if revision <= minRevision {
		return nil, revision, ErrMemberRevisionTooOld
	}

	pr, pw := io.Pipe()
	go func() {
		// Get the snapshot object.
		snapshot := c.server.Server.Backend().Snapshot()

		// Forward the snapshot to the pipe.
		n, err := snapshot.WriteTo(pw)
		if err != nil {
			log.WithError(err).Errorf("failed to write etcd snapshot out [written bytes: %d]", n)
		}
		pw.CloseWithError(err)

		if err := snapshot.Close(); err != nil {
			log.WithError(err).Errorf("failed to close etcd snapshot [written bytes: %d]", n)
		}
	}()

	return pr, revision, nil
}

func (c *Server) IsRunning() bool {
	return c.isRunning
}

func (c *Server) Stop(graceful, snapshot bool) {
	if !c.isRunning {
		return
	}
	if snapshot {
		if err := c.Snapshot(); err != nil {
			log.WithError(err).Error("failed to snapshot before graceful stop")
		}
	}
	if !graceful {
		c.server.Server.HardStop()
		c.server.Server = nil
	}
	c.server.Close()
	c.isRunning = false
	return
}

func (c *Server) startServer(ctx context.Context) error {
	var err error

	// Configure the server.
	etcdCfg := embed.NewConfig()
	etcdCfg.ClusterState = c.cfg.clusterState
	etcdCfg.Name = c.cfg.Name
	etcdCfg.Dir = c.cfg.DataDir
	etcdCfg.PeerAutoTLS = c.cfg.PeerSC.AutoTLS
	etcdCfg.PeerTLSInfo = c.cfg.PeerSC.TLSInfo()
	etcdCfg.ClientAutoTLS = c.cfg.ClientSC.AutoTLS
	etcdCfg.ClientTLSInfo = c.cfg.ClientSC.TLSInfo()
	etcdCfg.InitialCluster = initialCluster(c.cfg.initialPURLs)
	etcdCfg.LPUrls, _ = types.NewURLs([]string{peerURL(c.cfg.PrivateAddress, c.cfg.PeerSC.TLSEnabled())})
	etcdCfg.APUrls, _ = types.NewURLs([]string{peerURL(c.cfg.PrivateAddress, c.cfg.PeerSC.TLSEnabled())})
	etcdCfg.LCUrls, _ = types.NewURLs([]string{ClientURL(c.cfg.PrivateAddress, c.cfg.ClientSC.TLSEnabled())})
	etcdCfg.ACUrls, _ = types.NewURLs([]string{ClientURL(c.cfg.PublicAddress, c.cfg.ClientSC.TLSEnabled())})
	etcdCfg.ListenMetricsUrls = append(metricsURLs(c.cfg.PrivateAddress), metricsURLs("127.0.0.1")...)
	etcdCfg.Metrics = "extensive"
	etcdCfg.QuotaBackendBytes = c.cfg.DataQuota
	etcdCfg.AutoCompactionMode = c.cfg.AutoCompactionMode
	etcdCfg.AutoCompactionRetention = c.cfg.AutoCompactionRetention
	etcdCfg.ZapLoggerBuilder = logger.BuildZapConfigBuilder()
	etcdCfg.LogLevel = logger.GetZapLogLevelFromLogrus().String()

	// Start the server.
	c.server, err = embed.StartEtcd(etcdCfg)

	// Discard the gRPC logs, as the embed server will set that regardless of what was set before (i.e. at startup).
	etcdcl.SetLogger(grpclog.NewLoggerV2(ioutil.Discard, ioutil.Discard, os.Stderr))

	if err != nil {
		return fmt.Errorf("failed to start etcd: %s", err)
	}
	c.isRunning = true
	log.Infof("embedded etcd server is now running")

	// Wait until the server announces its ready, or until the start timeout is exceeded.
	//
	// When the server is joining an existing Client, it won't be until it has received a snapshot from healthy
	// members and sync'd from there.
	select {
	case <-c.server.Server.ReadyNotify():
		break
	case <-c.server.Err():
		// FIXME.
		panic("server failed to start, and continuing might stale the application, exiting instead (go.etcd.io/etcd/issues/9533)")
		c.Stop(false, false)
		return fmt.Errorf("server failed to start: %s", err)
	case <-ctx.Done():
		// FIXME.
		panic("server failed to start, and continuing might stale the application, exiting instead (go.etcd.io/etcd/issues/9533)")
		c.Stop(false, false)
		return fmt.Errorf("server took too long to become ready")
	}

	go c.runErrorWatcher()
	go c.runMemberCleaner()
	go c.runSnapshotter()

	return nil
}

func (c *Server) runErrorWatcher() {
	select {
	case <-c.server.Server.StopNotify():
		log.Warnf("etcd server is stopping")
		c.isRunning = false
		return
	case <-c.server.Err():
		log.Warnf("etcd server has crashed")
		c.Stop(false, false)
	}
}

func (c *Server) runMemberCleaner() {
	type memberT struct {
		name            string
		firstSeen       time.Time
		lastSeenHealthy time.Time
	}
	members := make(map[types.ID]*memberT)

	t := time.NewTicker(defaultMemberCleanerInterval)
	defer t.Stop()

	for {
		<-t.C
		if !c.IsRunning() {
			return
		}

		for _, member := range c.server.Server.Cluster().Members() {
			if !member.IsStarted() {
				continue
			}

			// Register the member's first seen time if it's a new member.
			if _, ok := members[member.ID]; !ok {
				members[member.ID] = &memberT{name: member.Name, firstSeen: time.Now()}
			}

			// Determine if the member is healthy and set the last time the member has been seen healthy.
			if c, err := NewClient([]string{URL2Address(member.PeerURLs[0])}, c.cfg.ClientSC, false); err == nil {
				if c.IsHealthy(5, 5*time.Second) {
					members[member.ID].lastSeenHealthy = time.Now()
				}
				c.Close()
			}
		}

		for id, member := range members {
			// Give the member time to start if it's a new one.
			if time.Since(member.firstSeen) < defaultStartTimeout && (member.lastSeenHealthy == time.Time{}) {
				continue
			}
			// Allow the member a graceful period.
			if time.Since(member.lastSeenHealthy) < c.cfg.UnhealthyMemberTTL {
				continue
			}
			log.Infof("removing member %q that's been unhealthy for %v", member.name, c.cfg.UnhealthyMemberTTL)

			cl, err := NewClient([]string{c.cfg.PrivateAddress}, c.cfg.ClientSC, false)
			if err != nil {
				log.WithError(err).Warn("failed to create etcd cluster client")
				continue
			}
			if err := cl.RemoveMember(member.name, uint64(id)); err == context.DeadlineExceeded {
				log.Warnf("failed to remove unhealthy member %q, it might be starting", member.name)
				continue
			} else if err != nil {
				log.WithError(err).Warnf("failed to remove unhealthy member %q", member.name)
				continue
			}

			delete(members, id)
		}
	}
}

func (c *Server) runSnapshotter() {
	if c.cfg.SnapshotProvider == nil || c.cfg.SnapshotInterval == 0 {
		log.Warn("periodic snapshots are disabled")
		return
	}

	t := time.NewTicker(c.cfg.SnapshotInterval)
	defer t.Stop()

	for {
		<-t.C
		if !c.IsRunning() {
			return
		}

		if err := c.Snapshot(); err != nil {
			log.WithError(err).Error("failed to snapshot")
		}
	}
}
