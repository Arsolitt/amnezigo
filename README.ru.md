# Amnezigo

**Amnezia** + **Go** = **Amnezigo** 🎮

CLI-утилита для генерации и управления конфигурациями [AmneziaWG](https://github.com/amnezia-vpn/amneziawg) v2.0.

## Возможности

- Генерация серверных конфигураций AmneziaWG с параметрами обфускации
- Управление клиентами (добавление, удаление, список, экспорт)
- Несколько протоколов обфускации (QUIC, DNS, DTLS, STUN)
- Автоматическое назначение IP-адресов клиентам
- Генерация правил iptables для NAT и проброса

## Установка

```bash
go install github.com/Arsolitt/amnezigo@latest
```

Или сборка из исходников:

```bash
git clone https://github.com/Arsolitt/amnezigo.git
cd amnezigo
go build -o amnezigo .
```

## Использование

### Инициализация сервера

```bash
amnezigo init --ipaddr 10.8.0.1/24
```

Параметры:
- `--ipaddr` - IP-адрес сервера с подсетью (обязательно)
- `--port` - Порт для прослушивания (по умолчанию: случайный 10000-65535)
- `--mtu` - Размер MTU (по умолчанию: 1280)
- `--dns` - DNS-серверы (по умолчанию: "1.1.1.1, 8.8.8.8")
- `--keepalive` - Интервал keepalive (по умолчанию: 25)
- `--protocol` - Протокол обфускации: random, quic, dns, dtls, stun (по умолчанию: random)
- `--client-to-client` - Разрешить трафик между клиентами
- `--iface` - Основной сетевой интерфейс (по умолчанию: автоопределение)

### Добавление клиента

```bash
amnezigo add laptop
amnezigo add phone --ipaddr 10.8.0.50
```

### Список клиентов

```bash
amnezigo list
```

### Экспорт конфигурации клиента

```bash
# Экспорт одного клиента
amnezigo export laptop --endpoint 1.2.3.4:55424

# Экспорт всех клиентов
amnezigo export --endpoint 1.2.3.4:55424
```

Параметры:
- `--endpoint` - Адрес сервера (по умолчанию: автоопределение внешнего IP)

### Удаление клиента

```bash
amnezigo remove laptop
```

## Файлы конфигурации

### Конфиг сервера (awg0.conf)

```ini
[Interface]
PrivateKey = <приватный-ключ-сервера>
Address = 10.8.0.1/24
ListenPort = 55424
MTU = 1280
PostUp = iptables -t nat ...
PostDown = iptables -t nat ...
Jc = 3
Jmin = 50
Jmax = 1000
S1 = 15
S2 = 16
S3 = 45
S4 = 10
H1 = 1827682742
H2 = 742172841
H3 = 1928417263
H4 = 281746291
I1 = <b 0xc0000000><r 16>
I2 = <b 0x40000000><r 12>
I3 = <b 0x80000000><t>
I4 = <b 0xc0000000><c>
I5 = <r 8>

[Peer]
#_Name = laptop
#_PrivateKey = <приватный-ключ-клиента>
PublicKey = <публичный-ключ-клиента>
AllowedIPs = 10.8.0.2/32
```

### Конфиг клиента (laptop.conf)

```ini
[Interface]
PrivateKey = <приватный-ключ-клиента>
Address = 10.8.0.2/32
DNS = 1.1.1.1, 8.8.8.8
MTU = 1280
Jc = 3
Jmin = 50
Jmax = 1000
...

[Peer]
PublicKey = <публичный-ключ-сервера>
PresharedKey = <psk>
Endpoint = 1.2.3.4:55424
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
```

## Параметры обфускации

AmneziaWG использует несколько параметров для обфускации трафика WireGuard:

- **Jc, Jmin, Jmax** - Параметры мусорных пакетов
- **S1-S4** - Размерные префиксы
- **H1-H4** - Значения заголовков (непересекающиеся области uint32)
- **I1-I5** - Custom Packet Strings (CPS) на основе шаблона протокола
