package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var (
	edgeIPAddr   string
	edgeProtocol string
)

// NewEdgeCommand creates the edge command group.
func NewEdgeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Manage edge server peers",
	}

	cmd.AddCommand(NewEdgeAddCommand())
	cmd.AddCommand(NewEdgeListCommand())
	cmd.AddCommand(NewEdgeExportCommand())
	cmd.AddCommand(NewEdgeRemoveCommand())

	return cmd
}

// NewEdgeAddCommand creates the edge add subcommand.
func NewEdgeAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new edge server to the configuration",
		Long: `Add a new edge server to the AmneziaWG hub configuration.

Generates a keypair for the edge and adds it to the server's edge list.
IP address can be auto-assigned or manually specified.

Example:
  amnezigo edge add moscow
  amnezigo edge add berlin --ipaddr 10.8.0.50
`,
		Args: cobra.ExactArgs(1),
		RunE: runEdgeAdd,
	}
	cmd.Flags().StringVar(&edgeIPAddr, "ipaddr", "", "Edge IP address (e.g., 10.8.0.5)")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runEdgeAdd(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	edge, err := mgr.AddEdge(args[0], edgeIPAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Edge '%s' added successfully\n", edge.Name)
	fmt.Printf("  IP Address: %s\n", edge.AllowedIPs)
	fmt.Printf("  Public Key: %s\n", edge.PublicKey)

	return nil
}

// NewEdgeListCommand creates the edge list subcommand.
func NewEdgeListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured edge servers",
		Long: `List all edge servers configured in the AmneziaWG server configuration.

Displays a table with edge name, IP address, and creation time.

Example:
  amnezigo edge list
`,
		RunE: runEdgeList,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runEdgeList(_ *cobra.Command, _ []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	edges := mgr.ListEdges()
	if len(edges) == 0 {
		fmt.Println("No edges configured")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(writer, "NAME\tIP\tCREATED")
	fmt.Fprintln(writer, strings.Repeat("-", separatorWidth))

	for _, edge := range edges {
		timestamp := ""
		if !edge.CreatedAt.IsZero() {
			timestamp = edge.CreatedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\n", edge.Name, edge.AllowedIPs, timestamp)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}
	return nil
}

// NewEdgeExportCommand creates the edge export subcommand.
func NewEdgeExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export edge server configuration",
		Long: `Export AWG configuration for the specified edge server.

The exported config is a client-style config where the edge connects to the hub.

Example:
  amnezigo edge export moscow
  amnezigo edge export --protocol quic moscow
`,
		Args: cobra.ExactArgs(1),
		RunE: runEdgeExport,
	}
	cmd.Flags().StringVar(&edgeProtocol, "protocol", "random", "Obfuscation protocol")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runEdgeExport(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	serverCfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	endpoint := resolveExportEndpoint(serverCfg)

	data, err := mgr.ExportEdge(args[0], edgeProtocol, endpoint)
	if err != nil {
		return err
	}

	configPath := args[0] + ".conf"
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write edge config: %w", err)
	}

	fmt.Printf("Exported edge '%s' to %s\n", args[0], configPath)
	return nil
}

// NewEdgeRemoveCommand creates the edge remove subcommand.
func NewEdgeRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an edge server from the configuration",
		Long: `Remove an edge server from the AmneziaWG hub configuration.

Example:
  amnezigo edge remove moscow
`,
		Args: cobra.ExactArgs(1),
		RunE: runEdgeRemove,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runEdgeRemove(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	if err := mgr.RemoveEdge(args[0]); err != nil {
		return err
	}
	fmt.Printf("Edge '%s' removed successfully\n", args[0])
	return nil
}
