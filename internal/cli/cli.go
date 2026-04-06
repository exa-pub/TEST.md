package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/testmd/testmd/internal/models"
	"github.com/testmd/testmd/internal/parser"
	"github.com/testmd/testmd/internal/patterns"
	"github.com/testmd/testmd/internal/report"
	"github.com/testmd/testmd/internal/resolver"
	"github.com/testmd/testmd/internal/state"
)

type context struct {
	root        string
	instances   []*models.TestInstance
	state       *models.State
	sourceFiles map[string]bool // set of TEST.md file paths
}

// Run executes the CLI.
func Run() {
	rootCmd := &cobra.Command{
		Use:           "testmd",
		Short:         "Track manual tests in markdown",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	var testmdPath string
	rootCmd.PersistentFlags().StringVar(&testmdPath, "testmd", "", "Path to TEST.md or its directory")

	rootCmd.AddCommand(
		statusCmd(&testmdPath),
		resolveCmd(&testmdPath),
		failCmd(&testmdPath),
		getCmd(&testmdPath),
		gcCmd(&testmdPath),
		ciCmd(&testmdPath),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func statusCmd(testmdPath *string) *cobra.Command {
	var reportMD, reportJSON string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*testmdPath)
			if err != nil {
				return err
			}
			results := resolver.ComputeStatuses(ctx.instances, ctx.state)
			report.PrintStatus(results)
			if reportMD != "" {
				if err := report.WriteReportMD(results, reportMD); err != nil {
					return err
				}
			}
			if reportJSON != "" {
				if err := report.WriteReportJSON(results, reportJSON); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&reportMD, "report-md", "", "Save markdown report")
	cmd.Flags().StringVar(&reportJSON, "report-json", "", "Save JSON report")
	return cmd
}

func resolveCmd(testmdPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <id>",
		Short: "Mark test(s) as resolved",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*testmdPath)
			if err != nil {
				return err
			}
			matches := resolver.FindInstances(ctx.instances, args[0])
			if len(matches) == 0 {
				return fmt.Errorf("no test matching '%s'", args[0])
			}
			for _, inst := range matches {
				resolver.ResolveTest(ctx.state, inst)
				suffix := labelSuffix(inst.Labels)
				fmt.Printf("Resolved: %s%s\n", inst.Definition.Title, suffix)
			}
			return save(ctx)
		},
	}
}

func failCmd(testmdPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "fail <id> <message>",
		Short: "Mark test as failed with a message",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*testmdPath)
			if err != nil {
				return err
			}
			matches := resolver.FindInstances(ctx.instances, args[0])
			if len(matches) == 0 {
				return fmt.Errorf("no test matching '%s'", args[0])
			}
			for _, inst := range matches {
				resolver.FailTest(ctx.state, inst, args[1])
				suffix := labelSuffix(inst.Labels)
				fmt.Printf("Failed: %s%s\n", inst.Definition.Title, suffix)
				fmt.Printf("  Message: %s\n", args[1])
			}
			return save(ctx)
		},
	}
}

func getCmd(testmdPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show test details and description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*testmdPath)
			if err != nil {
				return err
			}
			matches := resolver.FindInstances(ctx.instances, args[0])
			if len(matches) == 0 {
				return fmt.Errorf("no test matching '%s'", args[0])
			}
			results := resolver.ComputeStatuses(matches, ctx.state)
			for i, r := range results {
				report.PrintGet(r)
				if i < len(results)-1 {
					fmt.Println()
				}
			}
			return nil
		},
	}
}

func gcCmd(testmdPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "gc",
		Short: "Remove orphaned test records",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*testmdPath)
			if err != nil {
				return err
			}
			n := resolver.GCState(ctx.state, ctx.instances)
			if err := save(ctx); err != nil {
				return err
			}
			fmt.Printf("Removed %d orphaned record(s).\n", n)
			return nil
		},
	}
}

func ciCmd(testmdPath *string) *cobra.Command {
	var reportMD, reportJSON string
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "Check all tests pass (for CI). Exits 1 if any test needs attention",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*testmdPath)
			if err != nil {
				return err
			}
			results := resolver.ComputeStatuses(ctx.instances, ctx.state)

			if reportMD != "" {
				if err := report.WriteReportMD(results, reportMD); err != nil {
					return err
				}
			}
			if reportJSON != "" {
				if err := report.WriteReportJSON(results, reportJSON); err != nil {
					return err
				}
			}

			var failing []models.StatusResult
			for _, r := range results {
				if r.Status != "resolved" {
					failing = append(failing, r)
				}
			}

			if len(failing) == 0 {
				color.Green("OK: all tests resolved")
				return nil
			}

			bold := color.New(color.FgRed, color.Bold)
			bold.Printf("FAIL: %d test(s) require attention\n\n", len(failing))

			for _, r := range failing {
				s := map[string]struct {
					icon string
					c    *color.Color
				}{
					"failed":   {"✗", color.New(color.FgRed)},
					"outdated": {"⟳", color.New(color.FgYellow)},
					"pending":  {"…", color.New(color.FgCyan)},
				}[r.Status]
				suffix := labelSuffix(r.Instance.Labels)
				fmt.Printf("  %s  %s  %s%s  %s\n",
					s.c.Sprint(s.icon), r.Instance.ID,
					r.Instance.Definition.Title, suffix,
					s.c.Sprint(r.Status))
			}
			os.Exit(1)
			return nil
		},
	}
	cmd.Flags().StringVar(&reportMD, "report-md", "", "Save markdown report")
	cmd.Flags().StringVar(&reportJSON, "report-json", "", "Save JSON report")
	return cmd
}

// --- path resolution ---

func findTestMDUpward() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "TEST.md")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no TEST.md found (searched from cwd to filesystem root)")
		}
		dir = parent
	}
}

func resolvePath(testmdPath string) (testFile, root string, err error) {
	if testmdPath == "" {
		testFile, err = findTestMDUpward()
		if err != nil {
			return "", "", err
		}
		return testFile, filepath.Dir(testFile), nil
	}

	abs, err := filepath.Abs(testmdPath)
	if err != nil {
		return "", "", err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", "", fmt.Errorf("path not found: %s", testmdPath)
	}

	if info.IsDir() {
		testFile = filepath.Join(abs, "TEST.md")
		if _, err := os.Stat(testFile); err != nil {
			return "", "", fmt.Errorf("no TEST.md in %s", abs)
		}
		return testFile, abs, nil
	}

	return abs, filepath.Dir(abs), nil
}

// --- load / save ---

func load(testmdPath string) (*context, error) {
	testFile, root, err := resolvePath(testmdPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		return nil, err
	}

	fm, defs, err := parser.Parse(string(data), testFile)
	if err != nil {
		return nil, err
	}

	// Handle includes
	for _, inc := range fm.Include {
		incFile, err := filepath.Abs(filepath.Join(filepath.Dir(testFile), inc))
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(incFile); err != nil {
			return nil, fmt.Errorf("included file not found: %s", inc)
		}
		incData, err := os.ReadFile(incFile)
		if err != nil {
			return nil, err
		}
		incFM, incDefs, err := parser.Parse(string(incData), incFile)
		if err != nil {
			return nil, err
		}
		if len(incFM.Include) > 0 {
			return nil, fmt.Errorf("nested includes are not supported: %s includes %v", inc, incFM.Include)
		}
		defs = append(defs, incDefs...)
	}

	ignorefile := fm.Ignorefile
	if ignorefile == "" {
		ignorefile = ".gitignore"
	}
	ig := patterns.LoadIgnorefile(root, ignorefile)

	instances, err := resolver.BuildInstances(root, defs, ig)
	if err != nil {
		return nil, err
	}

	// Load and merge state from all source files
	sourceFiles := map[string]bool{testFile: true}
	for _, d := range defs {
		sourceFiles[d.SourceFile] = true
	}

	st := &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}
	for sf := range sourceFiles {
		fileSt, err := state.Load(sf)
		if err != nil {
			return nil, err
		}
		for id, rec := range fileSt.Tests {
			st.Tests[id] = rec
		}
	}

	return &context{
		root:        root,
		instances:   instances,
		state:       st,
		sourceFiles: sourceFiles,
	}, nil
}

func save(ctx *context) error {
	// Group test IDs by source file
	idsByFile := map[string]map[string]bool{}
	for sf := range ctx.sourceFiles {
		idsByFile[sf] = map[string]bool{}
	}
	for _, inst := range ctx.instances {
		sf := inst.Definition.SourceFile
		if _, ok := idsByFile[sf]; ok {
			idsByFile[sf][inst.ID] = true
		}
	}

	for sf, ids := range idsByFile {
		fileSt := &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}
		for id := range ids {
			if rec, ok := ctx.state.Tests[id]; ok {
				fileSt.Tests[id] = rec
			}
		}
		if len(fileSt.Tests) > 0 {
			if err := state.Save(sf, fileSt); err != nil {
				return err
			}
		} else {
			if err := state.StripBlock(sf); err != nil {
				return err
			}
		}
	}
	return nil
}

func labelSuffix(labels map[string]string) string {
	s := report.FormatLabels(labels)
	if s == "" {
		return ""
	}
	return " (" + s + ")"
}

func init() {
	// Disable cobra completions command
	cobra.EnableCommandSorting = false
}
