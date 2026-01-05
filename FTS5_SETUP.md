# Enabling FTS5 Support

FTS5 (Full-Text Search 5) is not enabled by default in many SQLite builds. Here's how to enable it:

## Why FTS5 Might Not Work

The `go-sqlite3` package links against the system SQLite library. If that library wasn't compiled with FTS5 support, you'll get the "no such module: fts5" error.

## Solutions

### Option 1: Build with FTS5 Tag (Recommended)

When building the Go application, use the `sqlite_fts5` build tag:

```bash
go build -tags sqlite_fts5 -o wikipedia_sqlite .
```

Or when running:

```bash
go run -tags sqlite_fts5 .
```

This tells `go-sqlite3` to use its embedded SQLite with FTS5 enabled.

### Option 2: Use System SQLite with FTS5

If you want to use the system SQLite, you need to ensure it's compiled with FTS5:

**On Linux (Ubuntu/Debian):**

```bash
# Install SQLite development files
sudo apt-get install libsqlite3-dev

# Check if FTS5 is available
sqlite3 :memory: "PRAGMA compile_options;" | grep -i fts5
```

**On Alpine (Docker):**
The default Alpine SQLite package may not have FTS5. You may need to compile SQLite from source or use a different base image.

**Compile SQLite with FTS5:**

```bash
wget https://www.sqlite.org/2024/sqlite-autoconf-*.tar.gz
tar -xzf sqlite-autoconf-*.tar.gz
cd sqlite-autoconf-*
./configure --enable-fts5
make
sudo make install
```

### Option 3: Check Current SQLite Build

You can check what features are available in your SQLite:

```bash
sqlite3 :memory: "PRAGMA compile_options;"
```

Look for `ENABLE_FTS5` in the output. If it's not there, FTS5 is not available.

### Docker Build

The Dockerfile has been updated to use the `sqlite_fts5` build tag. If you're building manually, make sure to use:

```bash
docker build --build-arg BUILD_TAGS=sqlite_fts5 .
```

Or update the Dockerfile to always use the tag (as we've done).

## Verification

After building with FTS5 support, you should see in the logs:

```
FTS5 is available in SQLite build
Using fts5 for full-text search
```

Instead of:

```
WARNING: FTS5 is not available in this SQLite build.
FTS5 not available, trying FTS4: ...
```

## Performance Note

FTS5 is generally faster and more feature-rich than FTS4, but FTS4 will work fine for most use cases. The application automatically falls back to FTS4 or LIKE queries if FTS5 is not available.
