//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	// confPathInNode is where the harness installs a config inside a node.
	confPathInNode = "/etc/awg0.conf"
	// daemonLogPath is the amneziawg-go log captured inside a node.
	daemonLogPath = "/tmp/amneziawg-go.log"
	// uapiSocketPath is the UAPI socket amneziawg-go creates for awg0.
	uapiSocketPath = "/var/run/amneziawg/awg0.sock"
	// tunnelDevice is the interface name used by every scenario.
	tunnelDevice = "awg0"
)

// dumpFieldNames maps the first record of `awg show awg0 dump` to field names,
// in the exact order dump_print emits them (amneziawg-tools/src/show.c).
var dumpFieldNames = []string{
	"private_key", "public_key", "listen_port",
	"jc", "jmin", "jmax",
	"s1", "s2", "s3", "s4",
	"h1", "h2", "h3", "h4",
	"i1", "i2", "i3", "i4", "i5",
	"header_protection_key",
	"content_padding_addition",
	"rekey_after_time",
	"rekey_timeout",
	"reject_after_time",
	"keepalive_timeout",
	"max_handshake_attempts",
	"random_trailers",
	"disable_cookies",
	"fwmark",
}

// interfaceHostKeys are the keys setconf does not accept in [Interface];
// they are stripped before a generated config is applied to a node.
var interfaceHostKeys = map[string]bool{
	"address":             true,
	"mtu":                 true,
	"dns":                 true,
	"publickey":           true,
	"postup":              true,
	"postdown":            true,
	"persistentkeepalive": true,
}

// runCommand runs bin with args and returns the combined output. dir and stdin
// are optional. A context timeout bounds the process.
func runCommand(bin string, args []string, dir string, stdin []byte, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	text := strings.TrimSpace(out.String())
	if ctx.Err() == context.DeadlineExceeded {
		return text, fmt.Errorf("%s %s: timed out after %s", bin, strings.Join(args, " "), timeout)
	}
	if err != nil {
		return text, fmt.Errorf("%s %s: %w", bin, strings.Join(args, " "), err)
	}
	return text, nil
}

// dockerRun runs one docker CLI command with the shared timeout.
func dockerRun(args ...string) (string, error) {
	return runCommand("docker", args, "", nil, dockerTimeout)
}

// netCounter makes docker network names unique within a test binary.
var netCounter atomic.Int64

// newNet creates an isolated docker network and removes it (with any leftover
// containers) at test cleanup.
func newNet(t *testing.T) string {
	t.Helper()
	name := fmt.Sprintf("amnezigo-e2e-%d-%d", os.Getpid(), netCounter.Add(1))
	if out, err := dockerRun("network", "create", name); err != nil {
		t.Fatalf("create docker network %s: %v\n%s", name, err, out)
	}
	t.Cleanup(func() {
		if out, err := dockerRun("ps", "-aq", "--filter", "network="+name); err == nil {
			for _, id := range strings.Fields(out) {
				_, _ = dockerRun("rm", "-f", id)
			}
		}
		_, _ = dockerRun("network", "rm", name)
	})
	return name
}

// node is a container running the e2e test image.
type node struct {
	name string
	ip   string
}

// startNode starts a long-lived container attached to the network and returns
// its name and bridge IP. The container is removed at test cleanup.
func startNode(t *testing.T, network, name string) node {
	t.Helper()
	fullName := network + "-" + name
	args := []string{
		"run", "-d", "--name", fullName, "--network", network,
		"--cap-add=NET_ADMIN", "--cap-add=NET_RAW", "--device", "/dev/net/tun",
		imageTag,
	}
	if out, err := dockerRun(args...); err != nil {
		t.Fatalf("start node %s: %v\n%s", fullName, err, out)
	}
	t.Cleanup(func() {
		_, _ = dockerRun("rm", "-f", fullName)
	})

	ip, err := dockerRun("inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", fullName)
	if err != nil {
		t.Fatalf("inspect node %s: %v", fullName, err)
	}
	if net.ParseIP(ip) == nil {
		t.Fatalf("node %s has unexpected IP %q", fullName, ip)
	}
	return node{name: fullName, ip: ip}
}

// execIn runs a shell script inside n and returns the combined output.
func execIn(t *testing.T, n node, script string) (string, error) {
	t.Helper()
	return dockerRun("exec", n.name, "sh", "-c", script)
}

// mustExec runs a shell script inside n and fails the test on error.
func mustExec(t *testing.T, n node, script string) string {
	t.Helper()
	out, err := execIn(t, n, script)
	if err != nil {
		t.Fatalf("exec in %s: %s: %v\n%s", n.name, script, err, out)
	}
	return out
}

// waitFor polls cond until it returns true or the timeout elapses.
func waitFor(cond func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// writeFileInNode copies data into path inside n.
func writeFileInNode(t *testing.T, n node, path string, data []byte) {
	t.Helper()
	args := []string{"exec", "-i", n.name, "sh", "-c", "cat > " + path}
	if out, err := runCommand("docker", args, "", data, dockerTimeout); err != nil {
		t.Fatalf("write %s in %s: %v\n%s", path, n.name, err, out)
	}
}

// generate runs the built CLI against manifest in a temp project directory and
// returns the server and client configs it wrote.
func generate(t *testing.T, manifest string) (serverConf, clientConf []byte) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "amnezigo.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if out, err := runCommand(cliPath, []string{"generate", "--project", dir}, "", nil, dockerTimeout); err != nil {
		t.Fatalf("amnezigo generate: %v\n%s", err, out)
	}
	read := func(peer string) []byte {
		path := filepath.Join(dir, "output", peer, "awg0.conf")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return data
	}
	return read("server"), read("client")
}

// confInterfaceValue returns the value of key in the [Interface] section of a
// generated config, or "" when the key is absent.
func confInterfaceValue(t *testing.T, conf []byte, key string) string {
	t.Helper()
	for _, raw := range strings.Split(string(conf), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "[") {
			if line != "[Interface]" {
				break
			}
			continue
		}
		field, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(field) != key {
			continue
		}
		return strings.TrimSpace(value)
	}
	return ""
}

// mustConfValue returns an [Interface] key value and fails when it is missing.
func mustConfValue(t *testing.T, conf []byte, key string) string {
	t.Helper()
	value := confInterfaceValue(t, conf, key)
	if value == "" {
		t.Fatalf("generated config has no %s in [Interface]:\n%s", key, conf)
	}
	return value
}

// prepareConf strips the [Interface] keys that setconf rejects, rewrites the
// [Peer] endpoint host to peerIP and returns the result. Everything else,
// including [Peer] PublicKey and the I1-I5 values, is preserved verbatim.
func prepareConf(t *testing.T, conf []byte, peerIP string) string {
	t.Helper()
	var out []string
	section := ""
	for _, raw := range strings.Split(string(conf), "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = strings.ToLower(strings.TrimSpace(trimmed[1 : len(trimmed)-1]))
			out = append(out, line)
			continue
		}
		field, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			out = append(out, line)
			continue
		}
		field = strings.ToLower(strings.TrimSpace(field))
		switch {
		case section == "interface" && interfaceHostKeys[field]:
			continue
		case section == "peer" && field == "endpoint":
			_, port, err := net.SplitHostPort(strings.TrimSpace(value))
			if err != nil {
				t.Fatalf("rewrite endpoint %q from generated config: %v", strings.TrimSpace(value), err)
			}
			out = append(out, fmt.Sprintf("Endpoint = %s:%s", peerIP, port))
		default:
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// startAWG launches amneziawg-go in the foreground with its log captured at
// daemonLogPath inside the node. The daemon outlives the docker exec.
func startAWG(t *testing.T, n node) {
	t.Helper()
	script := "amneziawg-go --foreground " + tunnelDevice + " >" + daemonLogPath + " 2>&1"
	if out, err := dockerRun("exec", "-d", n.name, "sh", "-c", script); err != nil {
		t.Fatalf("start amneziawg-go in %s: %v\n%s", n.name, err, out)
	}
}

// waitUAPISocket waits for the daemon's UAPI socket to appear.
func waitUAPISocket(t *testing.T, n node) {
	t.Helper()
	ok := waitFor(func() bool {
		_, err := execIn(t, n, "test -S "+uapiSocketPath)
		return err == nil
	}, 10*time.Second)
	if !ok {
		log, _ := execIn(t, n, "cat "+daemonLogPath+" 2>&1; ls -l /var/run/amneziawg 2>&1")
		t.Fatalf("UAPI socket %s did not appear in %s:\n%s", uapiSocketPath, n.name, log)
	}
}

// setconf applies an already-written config file and returns the raw error.
func setconf(t *testing.T, n node, path string) error {
	t.Helper()
	out, err := execIn(t, n, "awg setconf "+tunnelDevice+" "+path)
	if err != nil {
		return fmt.Errorf("awg setconf in %s: %v\n%s", n.name, err, out)
	}
	return nil
}

// awgLog returns the daemon log captured inside n ("" when unreadable).
func awgLog(t *testing.T, n node) string {
	t.Helper()
	out, err := execIn(t, n, "cat "+daemonLogPath+" 2>/dev/null")
	if err != nil {
		return ""
	}
	return out
}

// applyConf installs conf on n with the peer endpoint host rewritten to
// peerIP, starts the daemon, applies the config and brings the interface up
// with addr, mtu and the extra routes.
func applyConf(t *testing.T, n node, conf []byte, peerIP, addr string, mtu int, extraRoutes ...string) {
	t.Helper()
	prepared := []byte(prepareConf(t, conf, peerIP))
	writeFileInNode(t, n, confPathInNode, prepared)

	startAWG(t, n)
	waitUAPISocket(t, n)

	if err := setconf(t, n, confPathInNode); err != nil {
		t.Fatal(err)
	}
	mustExec(t, n, fmt.Sprintf("ip addr add %s dev %s", addr, tunnelDevice))
	mustExec(t, n, fmt.Sprintf("ip link set mtu %d dev %s", mtu, tunnelDevice))
	mustExec(t, n, "ip link set "+tunnelDevice+" up")
	for _, route := range extraRoutes {
		mustExec(t, n, fmt.Sprintf("ip route replace %s dev %s", route, tunnelDevice))
	}
}

// awgDump parses the interface record of `awg show awg0 dump` into a map keyed
// by the documented field names.
func awgDump(t *testing.T, n node) map[string]string {
	t.Helper()
	out := mustExec(t, n, "awg show "+tunnelDevice+" dump")
	line := strings.SplitN(strings.TrimSpace(out), "\n", 2)[0]
	fields := strings.Split(line, "\t")
	if len(fields) == len(dumpFieldNames)+1 && fields[0] == tunnelDevice {
		fields = fields[1:] // `awg show all dump` prefixes the interface name
	}
	if len(fields) < len(dumpFieldNames) {
		t.Fatalf("awg show %s dump on %s: got %d fields, want %d:\n%s",
			tunnelDevice, n.name, len(fields), len(dumpFieldNames), out)
	}
	dump := make(map[string]string, len(dumpFieldNames))
	for i, name := range dumpFieldNames {
		dump[name] = fields[i]
	}
	return dump
}

// latestHandshake returns the first peer's last handshake unix timestamp
// (0 when the peer never completed a handshake).
func latestHandshake(t *testing.T, n node) uint64 {
	t.Helper()
	out := mustExec(t, n, "awg show "+tunnelDevice+" latest-handshakes")
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		ts, err := strconv.ParseUint(strings.TrimSpace(fields[len(fields)-1]), 10, 64)
		if err != nil {
			t.Fatalf("parse latest-handshakes from %s: %q: %v", n.name, line, err)
		}
		return ts
	}
	return 0
}

// ping runs ping in n and returns an error including the captured output.
func ping(t *testing.T, n node, target string, extraArgs ...string) error {
	t.Helper()
	args := append([]string{"-c", "3", "-W", "5"}, extraArgs...)
	args = append(args, target)
	out, err := execIn(t, n, "ping "+strings.Join(args, " "))
	if err != nil {
		return fmt.Errorf("ping %s from %s: %v\n%s", target, n.name, err, out)
	}
	return nil
}

// mustPing runs ping in n and fails the test on error.
func mustPing(t *testing.T, n node, target string, extraArgs ...string) {
	t.Helper()
	if err := ping(t, n, target, extraArgs...); err != nil {
		t.Fatal(err)
	}
}

// captureInitPacket runs tcpdump in n until one UDP packet of exactly
// ipTotalLen bytes addressed to udpPort arrives, then returns that packet's
// UDP payload. trigger is called once the capture is listening.
func captureInitPacket(t *testing.T, n node, udpPort, ipTotalLen int, trigger func()) []byte {
	t.Helper()
	filter := fmt.Sprintf("udp and dst port %d and ip[2:2] = %d", udpPort, ipTotalLen)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "exec", n.name,
		"tcpdump", "-i", "eth0", "-c", "1", "-s", "0", "-w", "-", filter)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start tcpdump in %s: %v", n.name, err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	time.Sleep(time.Second) // let tcpdump open the capture socket
	trigger()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("tcpdump in %s did not capture a packet matching %q: %v\n%s",
				n.name, filter, err, stderr.String())
		}
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("tcpdump in %s timed out waiting for a packet matching %q:\n%s",
			n.name, filter, stderr.String())
	}

	payload, err := parseUDPPayload(stdout.Bytes())
	if err != nil {
		t.Fatalf("parse captured packet from %s: %v", n.name, err)
	}
	return payload
}

// parseUDPPayload extracts the UDP payload from the first record of a pcap
// stream captured on an Ethernet interface.
func parseUDPPayload(pcap []byte) ([]byte, error) {
	if len(pcap) < 24 {
		return nil, fmt.Errorf("pcap too short (%d bytes)", len(pcap))
	}
	var order binary.ByteOrder
	switch magic := binary.LittleEndian.Uint32(pcap[0:4]); magic {
	case 0xa1b2c3d4, 0xa1b23c4d:
		order = binary.LittleEndian
	case 0xd4c3b2a1, 0x4d3cb2a1:
		order = binary.BigEndian
	default:
		return nil, fmt.Errorf("unexpected pcap magic 0x%08x", magic)
	}
	if linkType := order.Uint32(pcap[20:24]); linkType != 1 {
		return nil, fmt.Errorf("unexpected pcap link type %d, want Ethernet", linkType)
	}
	if len(pcap) < 40 {
		return nil, fmt.Errorf("pcap has no packet record (%d bytes)", len(pcap))
	}
	inclLen := int(order.Uint32(pcap[32:36]))
	record := pcap[40:]
	if inclLen > len(record) {
		return nil, fmt.Errorf("truncated packet record: have %d bytes, want %d", len(record), inclLen)
	}
	record = record[:inclLen]

	const ethernetHeaderLen = 14
	if len(record) < ethernetHeaderLen {
		return nil, fmt.Errorf("truncated Ethernet frame (%d bytes)", len(record))
	}
	if etherType := binary.BigEndian.Uint16(record[12:14]); etherType != 0x0800 {
		return nil, fmt.Errorf("unexpected EtherType 0x%04x, want IPv4", etherType)
	}
	ip := record[ethernetHeaderLen:]
	if len(ip) < 20 {
		return nil, fmt.Errorf("truncated IPv4 header (%d bytes)", len(ip))
	}
	ihl := int(ip[0]&0x0f) * 4
	if ihl < 20 || len(ip) < ihl {
		return nil, fmt.Errorf("invalid IPv4 header length %d (packet %d bytes)", ihl, len(ip))
	}
	if proto := ip[9]; proto != 17 {
		return nil, fmt.Errorf("unexpected IP protocol %d, want UDP", proto)
	}
	udp := ip[ihl:]
	if len(udp) < 8 {
		return nil, fmt.Errorf("truncated UDP header (%d bytes)", len(udp))
	}
	udpLen := int(binary.BigEndian.Uint16(udp[4:6]))
	if udpLen < 8 || udpLen-8 > len(udp)-8 {
		return nil, fmt.Errorf("invalid UDP length %d (datagram %d bytes)", udpLen, len(udp))
	}
	return udp[8:udpLen], nil
}
