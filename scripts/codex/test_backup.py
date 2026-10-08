#!/usr/bin/env python3
"""Backup failure/recovery tests using isolated directories and a fake Docker CLI."""
import fcntl
import gzip
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
import unittest

SCRIPT = Path(__file__).with_name('backup.sh')
DOCKER = r'''#!/usr/bin/env python3
import hashlib, io, json, os, pathlib, shutil, sys, tarfile
root = pathlib.Path(os.environ['FAKE_ROOT'])
a = sys.argv[1:]
with (root / 'calls').open('a') as f:
    f.write(json.dumps(a) + '\n')
if a[0] == 'inspect':
    print('false' if os.environ.get('BACKEND_DOWN') and a[-1] == 'digidocs-backend-go' else 'true')
elif a[0] == 'volume':
    sys.exit(0)
elif a[0] == 'cp':
    shutil.copyfile(a[1], a[2].split(':', 1)[1])
elif a[0] == 'run':
    with tarfile.open(fileobj=sys.stdout.buffer, mode='w|gz') as archive:
        pass
elif a[0] == 'exec':
    c = a[2:]
    if c[0] == 'pg_dump':
        if os.environ.get('DUMP_FAIL'):
            print('partial dump')
            sys.exit(1)
        print('CREATE TABLE fixture (value text);')
        for n in range(100):
            print("INSERT INTO fixture VALUES ('" + hashlib.sha256(str(n).encode()).hexdigest() + "');")
    elif c[0] == 'sh':
        if os.environ.get('NO_CONFIG'): sys.exit(1)
        print(os.environ.get('TARGET', 'test-target'))
    elif c[0] == 'mktemp':
        p = root / 'payload'
        p.touch()
        print(p)
    elif c[0] == 'rm':
        pathlib.Path(c[-1]).unlink(missing_ok=True)
    elif c[0].endswith('digidocs-backup-sync'):
        op = c[1]
        if op == 'prune': sys.exit(0)
        local = pathlib.Path(c[2])
        remote = root / 'remote' / pathlib.Path(c[3]).name
        if op == 'verify':
            sys.exit(0 if remote.exists() and remote.read_bytes() == local.read_bytes() else 1)
        if op == 'upload':
            if os.environ.get('UPLOAD_FAIL'): sys.exit(1)
            shutil.copyfile(local, remote)
        else: sys.exit(2)
    else: sys.exit(2)
else: sys.exit(2)
'''


class BackupTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.backups = self.root / 'backups'
        self.backups.mkdir()
        (self.root / 'remote').mkdir()
        bindir = self.root / 'bin'
        bindir.mkdir()
        (bindir / 'docker').write_text(DOCKER)
        (bindir / 'docker').chmod(0o755)
        (bindir / 'date').write_text('#!/bin/sh\nprintf "%s\\n" "$TEST_STAMP"\n')
        (bindir / 'date').chmod(0o755)
        self.env = dict(os.environ, PATH=str(bindir) + ':' + os.environ['PATH'],
                        FAKE_ROOT=str(self.root), DIGIDOCS_BACKUP_DIR=str(self.backups),
                        TEST_STAMP='20261008-120000', DIGIDOCS_OFFSITE_REQUIRED='1',
                        BACKUP_KEEP_DAYS='14')

    def backup(self, name='db_test_20260901-030000.sql.gz', data=b'old backup'):
        p = self.backups / name
        p.write_bytes(gzip.compress(data))
        os.utime(p, (time.time() - 40 * 86400,) * 2)
        return p

    def run_backup(self, *args, **env):
        return subprocess.run(['bash', str(SCRIPT), *args], env=dict(self.env, **env),
                              text=True, capture_output=True, timeout=15)

    def calls(self):
        p = self.root / 'calls'
        return [json.loads(s) for s in p.read_text().splitlines()] if p.exists() else []

    def test_failed_sync_retains_old_and_new_then_recovers(self):
        old = self.backup()
        result = self.run_backup(UPLOAD_FAIL='1')
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertTrue(old.exists())
        self.assertEqual(len(list(self.backups.glob('db_*.sql.gz'))), 2)
        self.assertFalse(any('prune' in c for c in self.calls()))
        self.assertFalse(list((self.backups / '.offsite').glob('*.sha256')))
        result = self.run_backup(TEST_STAMP='20261008-120001')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(old.exists())
        self.assertTrue((self.root / 'remote' / old.name).exists())
        self.assertTrue(any('prune' in c for c in self.calls()))

    def test_sync_only_reuses_existing_copy_and_never_prunes(self):
        old = self.backup()
        (self.root / 'remote' / old.name).write_bytes(old.read_bytes())
        result = self.run_backup('--sync-only')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(old.exists())
        self.assertTrue((self.backups / '.offsite' / (old.name + '.sha256')).exists())
        self.assertFalse(any('upload' in c or 'prune' in c or 'pg_dump' in c for c in self.calls()))
        (self.root / 'calls').unlink()
        self.assertEqual(self.run_backup('--sync-only').returncode, 0)
        self.assertFalse(any('verify' in c or 'upload' in c for c in self.calls()))

    def test_changed_content_or_destination_invalidates_receipt(self):
        old = self.backup()
        self.assertEqual(self.run_backup('--sync-only').returncode, 0)
        old.write_bytes(gzip.compress(b'changed content'))
        self.assertEqual(self.run_backup('--sync-only').returncode, 0)
        self.assertEqual(old.read_bytes(), (self.root / 'remote' / old.name).read_bytes())
        (self.root / 'calls').unlink()
        self.assertEqual(self.run_backup('--sync-only', TARGET='new-target').returncode, 0)
        self.assertTrue(any('verify' in c for c in self.calls()))

    def test_missing_backend_or_config_fails_without_cleanup(self):
        old = self.backup()
        for env in ({'BACKEND_DOWN': '1'}, {'NO_CONFIG': '1'}):
            result = self.run_backup('--sync-only', **env)
            self.assertNotEqual(result.returncode, 0)
            self.assertTrue(old.exists())
            self.assertFalse(any('prune' in c for c in self.calls()))

    def test_failed_dump_never_publishes_partial_file(self):
        result = self.run_backup(DUMP_FAIL='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(list(self.backups.glob('*.gz')))
        self.assertFalse(list(self.backups.glob('*.partial')))

    def test_concurrent_execution_rejected(self):
        with (self.backups / '.backup.lock').open('w') as lock:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            result = self.run_backup('--sync-only')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('another backup/sync is running', result.stderr)
        self.assertEqual(self.calls(), [])

    def test_invalid_gzip_not_uploaded(self):
        old = self.backup()
        old.write_bytes(b'not gzip')
        result = self.run_backup('--sync-only')
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(old.exists())
        self.assertFalse(any('upload' in c for c in self.calls()))


if __name__ == '__main__':
    unittest.main()
