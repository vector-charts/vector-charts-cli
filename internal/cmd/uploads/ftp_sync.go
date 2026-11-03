package uploads

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/spf13/cobra"
	"github.com/zydro-marine/vector-charts-cli/internal/cli"
	"github.com/zydro-marine/vector-charts-cli/internal/httpx"
	"github.com/zydro-marine/vector-charts-cli/internal/output"
)

func newFTPSyncCmd(factory cli.Factory) *cobra.Command {
	var (
		dryRun       bool
		deleteRemote bool
		localDir     string
		ftpTarget    string
		username     string
		password     string
		prefix       string
	)
	cmd := &cobra.Command{
		Use:   "ftp-sync",
		Short: "Download chart files from an S-63 FTP provider and upload them",
		Long: `Interactively connect to an FTP chart provider, download the remote tree into a
local folder, then sync compatible files to Vector Charts uploads.

Prompts for FTP host/path, username, and password unless provided via flags.

The optional path is treated as relative to the FTP login directory (CWD into each
segment). Some providers reject absolute LIST paths.

Existing portal uploads are queried first; remote files whose sanitized filename
already exists are skipped (not downloaded again).

With --prefix, only files whose remote path contains that substring are considered.

Accepted download extensions match portal uploads: .000, .zip, .txt, .prm, .h5, .vcm.

With --dry-run, files that are not already uploaded are still downloaded locally,
but nothing is uploaded to the portal.

Deletions are opt-in: pass --delete to remove portal uploads missing from the FTP
set (default is not to delete).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			return runFTPSync(client, ftpSyncOptions{
				DryRun:       dryRun,
				DeleteRemote: deleteRemote,
				LocalDir:     localDir,
				FTPTarget:    ftpTarget,
				Username:     username,
				Password:     password,
				Prefix:       prefix,
			})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Download new files but do not upload to the portal")
	cmd.Flags().BoolVar(&deleteRemote, "delete", false, "Delete remote uploads not present on the FTP server (default: false)")
	cmd.Flags().StringVar(&localDir, "dir", "", "Local directory for FTP downloads (default: temp dir)")
	cmd.Flags().StringVar(&ftpTarget, "ftp", "", "FTP host[/path] relative to login dir (e.g. ftp.example.com/ENC)")
	cmd.Flags().StringVar(&username, "username", "", "FTP username")
	cmd.Flags().StringVar(&password, "password", "", "FTP password (prefer interactive prompt)")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Only include files whose remote path contains this substring")
	return cmd
}

type ftpSyncOptions struct {
	DryRun       bool
	DeleteRemote bool
	LocalDir     string
	FTPTarget    string
	Username     string
	Password     string
	Prefix       string
}

type ftpEndpoint struct {
	Addr string
	Path string // relative to login directory; empty means stay put
}

func runFTPSync(client *httpx.Client, opts ftpSyncOptions) error {
	target := strings.TrimSpace(opts.FTPTarget)
	if target == "" {
		var err error
		target, err = cli.Prompt("FTP host/path: ")
		if err != nil {
			return err
		}
		if target == "" {
			return fmt.Errorf("FTP host/path is required")
		}
	}

	username := strings.TrimSpace(opts.Username)
	if username == "" {
		var err error
		username, err = cli.Prompt("FTP username: ")
		if err != nil {
			return err
		}
		if username == "" {
			return fmt.Errorf("FTP username is required")
		}
	}

	password := opts.Password
	if password == "" {
		var err error
		password, err = cli.PromptSecret("FTP password: ")
		if err != nil {
			return err
		}
	}

	endpoint, err := parseFTPTarget(target)
	if err != nil {
		return err
	}

	dir := opts.LocalDir
	if dir == "" {
		dir, err = os.MkdirTemp("", "vc-ftp-sync-*")
		if err != nil {
			return err
		}
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	output.Printf("Fetching existing uploads …\n")
	uploadedNames, err := listUploadedFilenames(client)
	if err != nil {
		return err
	}
	output.Printf("Found %d existing upload filename(s)\n", len(uploadedNames))

	output.Printf("Connecting to %s …\n", endpoint.Addr)
	conn, err := ftp.Dial(endpoint.Addr, ftp.DialWithTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("ftp dial: %w", err)
	}
	defer conn.Quit()

	if err := conn.Login(username, password); err != nil {
		return fmt.Errorf("ftp login: %w", err)
	}

	loginDir, err := conn.CurrentDir()
	if err != nil {
		loginDir = "(unknown)"
	}
	output.Printf("Login directory: %s\n", loginDir)

	if endpoint.Path != "" {
		for _, seg := range strings.Split(endpoint.Path, "/") {
			if seg == "" || seg == "." {
				continue
			}
			if err := conn.ChangeDir(seg); err != nil {
				return fmt.Errorf("ftp cd %q (from %s): %w", seg, loginDir, err)
			}
		}
		cwd, err := conn.CurrentDir()
		if err == nil {
			output.Printf("Changed to %s\n", cwd)
		}
	}

	if opts.Prefix != "" {
		output.Printf("Filtering remote paths containing %q\n", opts.Prefix)
	}
	output.Printf("Scanning FTP tree into %s …\n", dir)
	stats, err := downloadFTPTree(conn, dir, uploadedNames, opts.Prefix)
	if err != nil {
		return err
	}
	output.Printf("FTP scan: downloaded=%d skipped_uploaded=%d skipped_prefix=%d seen=%d\n",
		stats.Downloaded, stats.SkippedUploaded, stats.SkippedPrefix, len(stats.SeenNames))

	if opts.DryRun {
		output.Printf("Dry run: syncing uploads without modifying the server\n")
	}
	return runSync(client, dir, syncOptions{
		DeleteRemote: opts.DeleteRemote,
		DryRun:       opts.DryRun,
		PresentNames: stats.SeenNames,
		DeletePrefix: opts.Prefix,
	})
}

func listUploadedFilenames(client *httpx.Client) (map[string]struct{}, error) {
	_, items, err := client.PortalListAll("/v1/charts/uploads", "uploads", 200)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		m, _ := item.(map[string]any)
		name := output.Str(m["filename"])
		if name == "" {
			continue
		}
		out[name] = struct{}{}
	}
	return out, nil
}

func parseFTPTarget(raw string) (ftpEndpoint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ftpEndpoint{}, fmt.Errorf("empty FTP target")
	}

	if !strings.Contains(raw, "://") {
		raw = "ftp://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ftpEndpoint{}, fmt.Errorf("invalid FTP target: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "" && scheme != "ftp" {
		return ftpEndpoint{}, fmt.Errorf("unsupported FTP scheme %q (use ftp://)", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return ftpEndpoint{}, fmt.Errorf("FTP host is required")
	}
	port := u.Port()
	if port == "" {
		port = "21"
	}

	// Keep path relative to the login directory. Absolute LIST/CWD paths are
	// rejected by some providers.
	p := strings.Trim(u.Path, "/")
	p = path.Clean(p)
	if p == "." {
		p = ""
	}

	return ftpEndpoint{
		Addr: net.JoinHostPort(host, port),
		Path: p,
	}, nil
}

type ftpDownloadStats struct {
	Downloaded      int
	SkippedUploaded int
	SkippedPrefix   int
	SeenNames       map[string]struct{}
}

func downloadFTPTree(conn *ftp.ServerConn, localDir string, uploadedNames map[string]struct{}, prefix string) (ftpDownloadStats, error) {
	stats := ftpDownloadStats{SeenNames: map[string]struct{}{}}
	err := walkFTP(conn, localDir, "", uploadedNames, prefix, &stats)
	return stats, err
}

func walkFTP(conn *ftp.ServerConn, localDir, relDir string, uploadedNames map[string]struct{}, prefix string, stats *ftpDownloadStats) error {
	entries, err := conn.List(".")
	if err != nil {
		cwd, _ := conn.CurrentDir()
		return fmt.Errorf("ftp list %s: %w", cwd, err)
	}

	for _, entry := range entries {
		name := entry.Name
		if name == "." || name == ".." {
			continue
		}
		localPath := filepath.Join(localDir, name)
		relPath := name
		if relDir != "" {
			relPath = path.Join(relDir, name)
		}

		switch entry.Type {
		case ftp.EntryTypeFolder:
			if strings.HasPrefix(name, ".") {
				continue
			}
			if err := os.MkdirAll(localPath, 0o755); err != nil {
				return err
			}
			if err := conn.ChangeDir(name); err != nil {
				return fmt.Errorf("ftp cd %q: %w", name, err)
			}
			err := walkFTP(conn, localPath, relPath, uploadedNames, prefix, stats)
			if cerr := conn.ChangeDirToParent(); cerr != nil && err == nil {
				err = fmt.Errorf("ftp cd ..: %w", cerr)
			}
			if err != nil {
				return err
			}
		case ftp.EntryTypeFile:
			ext := strings.ToLower(filepath.Ext(name))
			if _, ok := allowedExtensions[ext]; !ok {
				continue
			}
			if prefix != "" && !strings.Contains(relPath, prefix) {
				stats.SkippedPrefix++
				continue
			}
			key := sanitizeFilename(name)
			stats.SeenNames[key] = struct{}{}
			if _, exists := uploadedNames[key]; exists {
				stats.SkippedUploaded++
				output.Printf("  skip %s (already uploaded)\n", relPath)
				continue
			}
			if err := downloadFTPFile(conn, name, localPath); err != nil {
				return err
			}
			stats.Downloaded++
			output.Printf("  got %s\n", relPath)
		}
	}
	return nil
}

func downloadFTPFile(conn *ftp.ServerConn, remoteName, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	resp, err := conn.Retr(remoteName)
	if err != nil {
		return fmt.Errorf("ftp retr %s: %w", remoteName, err)
	}
	defer resp.Close()

	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp); err != nil {
		return fmt.Errorf("ftp write %s: %w", localPath, err)
	}
	return nil
}
