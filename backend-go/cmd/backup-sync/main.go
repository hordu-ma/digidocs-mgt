// Command backup-sync uploads (and prunes) backup files on the Synology NAS
// this deployment already trusts for object storage, reusing
// internal/storage/synology (the same client the API server uses for
// documents/data assets) instead of reimplementing DSM session auth and
// multipart upload in a shell script.
//
// It exists because scripts/codex/backup.sh's PostgreSQL/git-repo backups land
// on the same disk as the data they protect — a local disk failure destroys
// both the source and every backup at once. Running this from inside the
// backend-go container means it inherits the exact SYNOLOGY_* credentials and
// network path already proven to work for the app's own storage traffic.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"time"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/storage"
	"digidocs-mgt/backend-go/internal/storage/synology"
)

// objectStore is the slice of storage.Provider this tool actually needs.
// *synology.Provider satisfies it directly; tests supply a fake.
type objectStore interface {
	PutObject(ctx context.Context, input storage.PutObjectInput) (storage.PutObjectResult, error)
	GetObject(ctx context.Context, objectKey string) (*storage.GetObjectOutput, error)
	ListDir(ctx context.Context, folderPath string) ([]storage.FileInfo, error)
	DeleteObject(ctx context.Context, objectKey string) error
}

func main() {
	if err := run(os.Args[1:], config.Load, newSynologyStore); err != nil {
		log.Fatalf("[backup-sync] %v", err)
	}
}

func newSynologyStore(cfg config.Config) objectStore {
	return synology.NewProvider(synology.Config{
		Host:               cfg.SynologyHost,
		Port:               cfg.SynologyPort,
		HTTPS:              cfg.SynologyHTTPS,
		InsecureSkipVerify: cfg.SynologyInsecureSkipVerify,
		Account:            cfg.SynologyAccount,
		Password:           cfg.SynologyPassword,
		SharePath:          cfg.SynologySharePath,
	})
}

func run(args []string, loadConfig func() config.Config, newStore func(config.Config) objectStore) error {
	if len(args) == 0 {
		return errors.New("usage: backup-sync <upload|verify|prune> ...")
	}
	sub, rest := args[0], args[1:]

	cfg := loadConfig()
	if cfg.SynologyHost == "" {
		return errors.New("SYNOLOGY_HOST is not set; this container is not configured for Synology storage, cannot sync offsite")
	}
	store := newStore(cfg)

	switch sub {
	case "upload":
		return runUpload(rest, store)
	case "verify":
		return runVerify(rest, store)
	case "prune":
		return runPrune(rest, store)
	default:
		return fmt.Errorf("unknown subcommand %q, want upload, verify or prune", sub)
	}
}

// runUpload pushes one local file to objectKey, creating parent folders and
// overwriting any previous file at that key (backup filenames are
// timestamped, so a real collision would mean a bug, not a legitimate reuse).
func runUpload(args []string, store objectStore) error {
	if len(args) != 2 {
		return errors.New("usage: backup-sync upload <local-file> <remote-object-key>")
	}
	localPath, objectKey := args[0], args[1]

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()

	result, err := store.PutObject(context.Background(), storage.PutObjectInput{
		ObjectKey:   objectKey,
		Reader:      f,
		Overwrite:   true,
		CreatePaths: true,
	})
	if err != nil {
		return fmt.Errorf("upload %s to %s: %w", localPath, objectKey, err)
	}

	log.Printf("uploaded %s -> synology:%s", localPath, result.ObjectKey)
	return runVerify(args, store)
}

// runVerify streams the remote object back; a successful upload response alone
// does not prove that the retained backup contains the same bytes.
func runVerify(args []string, store objectStore) error {
	if len(args) != 2 {
		return errors.New("usage: backup-sync verify <local-file> <remote-object-key>")
	}
	local, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("open backup: %w", err)
	}
	defer local.Close()
	localHash := sha256.New()
	localSize, err := io.Copy(localHash, local)
	if err != nil {
		return fmt.Errorf("hash local backup: %w", err)
	}
	remote, err := store.GetObject(context.Background(), args[1])
	if err != nil {
		return fmt.Errorf("download backup for verification: %w", err)
	}
	defer remote.Reader.Close()
	remoteHash := sha256.New()
	remoteSize, err := io.Copy(remoteHash, remote.Reader)
	if err != nil {
		return fmt.Errorf("hash remote backup: %w", err)
	}
	if localSize != remoteSize || !bytes.Equal(localHash.Sum(nil), remoteHash.Sum(nil)) {
		return fmt.Errorf("backup verification failed for %s: local_bytes=%d remote_bytes=%d SHA-256 mismatch", args[1], localSize, remoteSize)
	}
	log.Printf("verified %s bytes=%d sha256=%x", args[1], localSize, localHash.Sum(nil))
	return nil
}

// runPrune deletes files under folder older than keepDays, mirroring
// backup.sh's local `find -mtime +KEEP_DAYS -delete` retention so the NAS
// side doesn't grow forever once offsite sync starts running daily.
var backupName = regexp.MustCompile(`^(db_[A-Za-z0-9_-]+_[0-9]{8}-[0-9]{6}\.sql\.gz|coderepos_[0-9]{8}-[0-9]{6}\.tar\.gz)$`)

func runPrune(args []string, store objectStore) error {
	if len(args) != 2 {
		return errors.New("usage: backup-sync prune <remote-folder> <keep-days>")
	}
	folder := args[0]
	keepDays, err := strconv.Atoi(args[1])
	if err != nil || keepDays < 0 {
		return fmt.Errorf("keep-days must be a non-negative integer, got %q", args[1])
	}

	entries, err := store.ListDir(context.Background(), folder)
	if err != nil {
		return fmt.Errorf("list %s: %w", folder, err)
	}

	cutoff := time.Now().Add(-time.Duration(keepDays) * 24 * time.Hour)
	deleted := 0
	for _, entry := range entries {
		if entry.IsDir || !backupName.MatchString(entry.Name) || entry.ModifiedAt.After(cutoff) {
			continue
		}
		key := path.Join(folder, entry.Name)
		if err := store.DeleteObject(context.Background(), key); err != nil {
			return fmt.Errorf("delete %s: %w", key, err)
		}
		log.Printf("pruned synology:%s (modified %s)", key, entry.ModifiedAt.Format(time.RFC3339))
		deleted++
	}
	log.Printf("prune complete: removed %d file(s) under synology:%s older than %d day(s)", deleted, folder, keepDays)
	return nil
}
