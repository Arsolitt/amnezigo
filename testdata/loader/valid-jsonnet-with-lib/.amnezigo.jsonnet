local net = import 'network.libsonnet';
{
  version: 1,
  network: net,
  obfuscation: {
    protocol: 'quic',
    s1: 30, s2: 35, s3: 20, s4: 12,
    h1: { min: 100, max: 5000000 },
    h2: { min: 10000000, max: 200000000 },
    h3: { min: 400000000, max: 800000000 },
    h4: { min: 1000000000, max: 2100000000 },
    jc: 5, jmin: 250, jmax: 750,
  },
  peers: {
    server: {
      address: '10.0.0.1/24',
      endpoint: 'vpn.example.com:51820',
      listen_port: 51820,
    },
  },
}
