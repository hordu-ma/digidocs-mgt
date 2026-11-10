// Command storage-audit reports storage objects that no database row
// references (orphans) and database references whose object is missing.
//
// It is strictly read-only: it lists the storage tree and queries the database,
// and never deletes or moves anything. Orphans come from overwritten
// same-name versions, interrupted uploads and removed rows; reviewing the
// report and deleting by hand is deliberate, because wrongly deleting a
// customer file costs far more than an orphan's disk space.
//
//	digidocs-storage-audit            # human-readable report
//	digidocs-storage-audit -json      # machine-readable report
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/db"
	"digidocs-mgt/backend-go/internal/storage"
	"digidocs-mgt/backend-go/internal/storage/synology"
)

// auditedPrefixes are the trees the application writes uploads into.
var auditedPrefixes = []string{"documents", "data"}

const maxDepth = 6

type lister interface {
	ListDir(ctx context.Context, folderPath string) ([]storage.FileInfo, error)
}

type object struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
}

type report struct {
	GeneratedAt    string   `json:"generated_at"`
	StoredObjects  int      `json:"stored_objects"`
	ReferencedKeys int      `json:"referenced_keys"`
	Orphans        []object `json:"orphans"`
	OrphanBytes    int64    `json:"orphan_bytes"`
	MissingObjects []string `json:"missing_objects"`
	ListErrors     []string `json:"list_errors,omitempty"`
}

func main() {
	asJSON := flag.Bool("json", false, "print the report as JSON")
	flag.Parse()
	cfg := config.Load()
	if cfg.StorageBackend != "synology" {
		log.Fatalf("[storage-audit] STORAGE_BACKEND=%q: only synology storage persists objects to audit", cfg.StorageBackend)
	}
	database, err := db.OpenPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[storage-audit] open database: %v", err)
	}
	defer database.Close()
	store := synology.NewProvider(synology.Config{
		Host: cfg.SynologyHost, Port: cfg.SynologyPort, HTTPS: cfg.SynologyHTTPS,
		InsecureSkipVerify: cfg.SynologyInsecureSkipVerify, Account: cfg.SynologyAccount,
		Password: cfg.SynologyPassword, SharePath: cfg.SynologySharePath,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	rep, err := audit(ctx, store, database, cfg.SynologySharePath)
	if err != nil {
		log.Fatalf("[storage-audit] %v", err)
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
		return
	}
	printReport(os.Stdout, rep)
}

func audit(ctx context.Context, store lister, database *sql.DB, sharePath string) (report, error) {
	stored, listErrors := walk(ctx, store, sharePath)
	referenced, err := referencedKeys(ctx, database)
	if err != nil {
		return report{}, err
	}
	rep := reconcile(stored, referenced)
	rep.ListErrors = listErrors
	return rep, nil
}

// walk lists every file under the audited prefixes, returning object keys
// relative to the share (the form stored in the database).
func walk(ctx context.Context, store lister, sharePath string) ([]object, []string) {
	var objects []object
	var errs []string
	var visit func(dir string, depth int)
	visit = func(dir string, depth int) {
		if depth > maxDepth {
			return
		}
		entries, err := store.ListDir(ctx, dir)
		if err != nil {
			errs = append(errs, dir+": "+err.Error())
			return
		}
		for _, e := range entries {
			key := relativeKey(e.Path, sharePath)
			if key == "" {
				key = strings.TrimPrefix(dir+"/"+e.Name, "/")
			}
			if e.IsDir {
				visit(key, depth+1)
				continue
			}
			objects = append(objects, object{Key: key, Size: e.Size})
		}
	}
	for _, prefix := range auditedPrefixes {
		visit(prefix, 0)
	}
	return objects, errs
}

func relativeKey(fullPath, sharePath string) string {
	p := strings.TrimPrefix(fullPath, strings.TrimRight(sharePath, "/")+"/")
	return strings.TrimPrefix(p, "/")
}

// referencedKeys includes soft-deleted rows: a deleted document can still be
// restored, so its files are not orphans.
func referencedKeys(ctx context.Context, database *sql.DB) (map[string]bool, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT storage_object_key FROM document_versions
		UNION
		SELECT storage_object_key FROM data_assets
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys[strings.TrimPrefix(k, "/")] = true
	}
	return keys, rows.Err()
}

func reconcile(stored []object, referenced map[string]bool) report {
	rep := report{GeneratedAt: time.Now().Format(time.RFC3339), StoredObjects: len(stored), ReferencedKeys: len(referenced), Orphans: []object{}, MissingObjects: []string{}}
	present := map[string]bool{}
	for _, o := range stored {
		present[o.Key] = true
		if !referenced[o.Key] {
			rep.Orphans = append(rep.Orphans, o)
			rep.OrphanBytes += o.Size
		}
	}
	for k := range referenced {
		inAudited := false
		for _, p := range auditedPrefixes {
			if strings.HasPrefix(k, p+"/") {
				inAudited = true
			}
		}
		if inAudited && !present[k] {
			rep.MissingObjects = append(rep.MissingObjects, k)
		}
	}
	sort.Slice(rep.Orphans, func(i, j int) bool { return rep.Orphans[i].Key < rep.Orphans[j].Key })
	sort.Strings(rep.MissingObjects)
	return rep
}

func printReport(w io.Writer, rep report) {
	fmt.Fprintf(w, "存储对账报告 %s（只读，未做任何删除）\n", rep.GeneratedAt)
	fmt.Fprintf(w, "存储中的文件：%d；数据库引用：%d\n", rep.StoredObjects, rep.ReferencedKeys)
	fmt.Fprintf(w, "\n孤儿文件（存储中有、数据库无引用）：%d 个，共 %d 字节\n", len(rep.Orphans), rep.OrphanBytes)
	for _, o := range rep.Orphans {
		fmt.Fprintf(w, "  %s  (%d B)\n", o.Key, o.Size)
	}
	fmt.Fprintf(w, "\n缺失文件（数据库引用、存储中找不到）：%d 个\n", len(rep.MissingObjects))
	for _, k := range rep.MissingObjects {
		fmt.Fprintf(w, "  %s\n", k)
	}
	if len(rep.ListErrors) > 0 {
		fmt.Fprintf(w, "\n列目录失败（结果可能不完整）：\n")
		for _, e := range rep.ListErrors {
			fmt.Fprintf(w, "  %s\n", e)
		}
	}
}
