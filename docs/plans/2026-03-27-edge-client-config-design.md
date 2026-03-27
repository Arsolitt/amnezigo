# Edge AWG Client Config Generation

Date: 2026-03-27

## Goal

Enable amnezigo to generate client-style AWG server configs — configs where the AWG interface connects to a remote peer (hub) instead of serving local clients. This is needed for edge servers that connect to the hub's AWG tunnel.

## Background

Currently `ServerConfig` and `BuildClientConfig` represent two distinct concepts:
- `ServerConfig`: AWG server with `[Interface]` + `[Peer]` sections. Used for hubs with multiple clients.
- `ClientConfig` (from `BuildClientConfig`): Client-side config for end-user devices (laptop, phone). Used for export.

The gap: an edge server needs a server-side AWG config that has exactly ONE peer (the hub) and no client peers. It's structurally similar to `ServerConfig` but semantically different — the edge is the one initiating the connection.

## Feature 1: Server-Side Client Config (Peer-to-Hub)

### Problem

Edge servers need an AWG config file (`awg0.conf`) that:
1. Defines the local interface with the edge's private key and IP address.
2. Defines exactly one peer — the hub — with the hub's public key, PSK, endpoint, and allowed IPs.
3. Does NOT define any client peers (no `[Peer]` sections for clients).

This is NOT the same as `ClientConfig` (which is for end-user devices and includes DNS, CPS obfuscation, etc.).

### Business Logic

1. New type `PeerToHubConfig` (or extend `ServerConfig` usage):
   - `Interface`: same as `InterfaceConfig` — private key, public key, address, listen port, MTU.
   - `Peers`: exactly one `PeerConfig` entry — the hub.
   - `Obfuscation`: server-side obfuscation params (same as current).
   - `Endpoint`: the hub's public endpoint (address:port).

2. The hub peer entry:
   - `PublicKey`: hub's AWG public key.
   - `PresharedKey`: shared PSK between edge and hub.
   - `AllowedIPs`: hub's subnet or `0.0.0.0/0` depending on routing strategy.
   - `Endpoint`: hub's public address:port.
   - `PersistentKeepalive`: 25 (to keep NAT mappings alive).

3. Config generation:
   - Uses existing `WriteServerConfig` — the INI format is the same.
   - The difference is semantic: caller ensures exactly one peer.

4. No changes to `BuildClientConfig` — that remains for end-user device exports.

### What changes

- No new types needed — `ServerConfig` with a single peer already represents this.
- Documentation: clarify that `ServerConfig` with `len(Peers) == 1` is the edge pattern.
- Possibly: `BuildEdgeConfig(hubPublicKey, hubPSK, hubEndpoint, hubAllowedIPs, edgeInterface)` convenience function.

## Feature 2: AllowedIPs Strategy for Hub Peer

### Problem

What should `AllowedIPs` be for the hub peer on an edge? This determines what traffic goes through the AWG tunnel.

### Business Logic

1. If edge is proxy-only (no TPROXY):
   - `AllowedIPs` = hub's AWG subnet only (e.g., `10.110.2.0/24`).
   - Edge only sends AWG control plane traffic through the tunnel.
   - All user traffic arrives via VLESS+Reality on the public interface.

2. If edge also does TPROXY (future):
   - `AllowedIPs` = `0.0.0.0/0` (all traffic through tunnel).
   - This would require TPROXY on the edge too.

3. For the target architecture (proxy-only edges):
   - Hub subnet only. This minimizes tunnel traffic to AWG management.

### What changes

- This is a caller decision, not an amnezigo change. The caller (cheburbox) sets `AllowedIPs` based on the role/config.

## Open Questions

1. Does the edge need PostUp/PostDown iptables rules? If no NAT/forwarding (proxy-only), probably not — but the edge needs `awg set fwmark` + `ip rule` for policy routing if it's routing specific traffic through the tunnel.
2. Should `BuildEdgeConfig` be a new function or should the caller just construct `ServerConfig` manually? Given it's a simple case, a convenience function makes sense.
