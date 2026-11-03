package uploads

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zydro-marine/vector-charts-cli/internal/cli"
	"github.com/zydro-marine/vector-charts-cli/internal/httpx"
	"github.com/zydro-marine/vector-charts-cli/internal/output"
)

var sanitizeFilenameRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func newSyncCmd(factory cli.Factory) *cobra.Command {
	var deleteRemote, dryRun bool
	cmd := &cobra.Command{
		Use:   "sync [path]",
		Short: "Upload local chart files missing remotely; optionally prune remote extras",
		Long: `Scan a local folder for chart files the portal accepts (.000, .zip, .txt, .prm, .h5, .vcm),
compare against existing uploads by sanitized filename, and upload any that are not already present.

With --delete, also remove remote uploads whose filename is not found locally.

Defaults to the current working directory when path is omitted. Scans recursively.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			return runSync(client, dir, syncOptions{DeleteRemote: deleteRemote, DryRun: dryRun})
		},
	}
	cmd.Flags().BoolVar(&deleteRemote, "delete", false, "Delete remote uploads not present locally (default: false)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be uploaded/deleted without changing the server")
	return cmd
}

type syncOptions struct {
	DeleteRemote bool
	DryRun       bool
	// PresentNames, when set, is the authoritative filename set for --delete
	// (e.g. all FTP filenames including ones skipped as already uploaded).
	PresentNames map[string]struct{}
	RemoteItems  []any
	// DeletePrefix, when set with PresentNames, limits --delete to remote
	// uploads whose filename contains this substring (avoids deleting
	// unrelated uploads when FTP sync used --prefix).
	DeletePrefix string
}

func runSync(client *httpx.Client, dir string, opts syncOptions) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	localByName, err := collectLocalFiles(dir)
	if err != nil {
		return err
	}

	items := opts.RemoteItems
	if items == nil {
		_, items, err = client.PortalListAll("/v1/charts/uploads", "uploads", 200)
		if err != nil {
			return err
		}
	}

	remoteByName := map[string][]map[string]any{}
	for _, item := range items {
		m, _ := item.(map[string]any)
		name := output.Str(m["filename"])
		if name == "" {
			continue
		}
		remoteByName[name] = append(remoteByName[name], m)
	}

	presentNames := opts.PresentNames
	if presentNames == nil {
		presentNames = make(map[string]struct{}, len(localByName))
		for name := range localByName {
			presentNames[name] = struct{}{}
		}
	}

	var uploaded, skipped, deleted, failed int

	for name, path := range localByName {
		if _, exists := remoteByName[name]; exists {
			skipped++
			output.Printf("skip  %s (already uploaded)\n", name)
			continue
		}
		if opts.DryRun {
			uploaded++
			output.Printf("would upload %s\n", path)
			continue
		}
		output.Printf("upload %s\n", path)
		out, err := client.UploadChart(path)
		if err != nil {
			failed++
			output.Errf("error: failed to upload %s: %v\n", path, err)
			continue
		}
		uploaded++
		id := output.Str(out["id"])
		status := output.ColorState(output.Str(out["status"]))
		if id != "" {
			output.Printf("  queued id=%s status=%s\n", id, status)
		}
	}

	if opts.DeleteRemote {
		for name, remotes := range remoteByName {
			if opts.DeletePrefix != "" && !strings.Contains(name, opts.DeletePrefix) {
				continue
			}
			if _, exists := presentNames[name]; exists {
				continue
			}
			for _, m := range remotes {
				id := output.Str(m["id"])
				if id == "" {
					continue
				}
				if opts.DryRun {
					deleted++
					output.Printf("would delete %s (%s)\n", name, id)
					continue
				}
				output.Printf("delete %s (%s)\n", name, id)
				if err := client.PortalJSON("DELETE", "/v1/charts/uploads/"+id, nil, nil); err != nil {
					failed++
					output.Errf("error: failed to delete %s: %v\n", id, err)
					continue
				}
				deleted++
			}
		}
	}

	label := "Sync complete"
	if opts.DryRun {
		label = "Dry run complete"
	}
	output.Printf("\n%s: uploaded=%d skipped=%d deleted=%d failed=%d local=%d remote=%d\n",
		label, uploaded, skipped, deleted, failed, len(localByName), len(items))
	if failed > 0 {
		return fmt.Errorf("%d sync operation(s) failed", failed)
	}
	return nil
}

// collectLocalFiles walks dir recursively and returns sanitized-basename -> path.
// Duplicate basenames keep the first path found and print a warning for later ones.
func collectLocalFiles(dir string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != dir && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if _, ok := allowedExtensions[ext]; !ok {
			return nil
		}
		key := sanitizeFilename(name)
		if prev, exists := out[key]; exists {
			output.Errf("warning: duplicate filename %q; keeping %s, ignoring %s\n", key, prev, path)
			return nil
		}
		out[key] = path
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Matches portal chartUploadUtils.sanitize.
func sanitizeFilename(name string) string {
	return sanitizeFilenameRe.ReplaceAllString(name, "_")
}
