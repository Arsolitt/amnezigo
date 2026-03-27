# Amnezigo

**Amnezia** + **Go** = **Amnezigo**

CLI-утилита и Go-библиотека для генерации и управления конфигурациями [AmneziaWG](https://github.com/amnezia-vpn/amneziawg) v2.0.

## Возможности

- Генерация серверных конфигураций AmneziaWG с параметрами обфускации
- Управление клиентами и edge-серверами: добавление, удаление, просмотр списка, экспорт
- 5 протоколов обфускации: QUIC, DNS, DTLS, STUN, Random
- Автоматическое назначение IP-адресов (клиенты и edge-серверы используют общий пул)
- Автоопределение эндпоинта (IPv4/IPv6) через icanhazip.com
- Генерация правил iptables для NAT и проброса трафика
- Параметры обфускации I1-I5 генерируются для каждого клиента при экспорте
- Динамическое переключение режима client-to-client
- Поддержка edge-серверов (топология hub-and-spoke)
- Использование как CLI-утилиты или Go-библиотеки

## Установка

### go install

```bash
go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest
```

### Сборка из исходников

```bash
git clone https://github.com/Arsolitt/amnezigo.git
cd amnezigo
go build -o build/amnezigo ./cmd/amnezigo/
```

### Docker

```bash
docker build -t amnezigo .
docker run --rm -v $(pwd):/data amnezigo init --ipaddr 10.8.0.1/24
```

## Быстрый старт

```bash
# 1. Инициализация сервера
amnezigo init --ipaddr 10.8.0.1/24

# 2. Добавление клиента
amnezigo client add laptop

# 3. Экспорт конфигурации клиента
amnezigo client export laptop
```

## Использование

### init — Инициализация сервера

```bash
amnezigo init --ipaddr 10.8.0.1/24
amnezigo init --ipaddr 10.8.0.1/24 --port 51820 --mtu 1420
amnezigo init --ipaddr 10.8.0.1/24 --endpoint-v4 1.2.3.4 --endpoint-v6 "[2001:db8::1]"
```

Флаги:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--ipaddr` | *(обязательно)* | IP-адрес сервера с подсетью (например, `10.8.0.1/24`) |
| `--port` | случайный 10000-65535 | Порт для прослушивания |
| `--mtu` | 1280 | Размер MTU |
| `--dns` | `1.1.1.1, 8.8.8.8` | DNS-серверы *(принимается, но не сохраняется)* |
| `--keepalive` | 25 | Интервал keepalive *(принимается, но не сохраняется)* |
| `--client-to-client` | false | Разрешить трафик между клиентами |
| `--iface` | автоопределение | Основной сетевой интерфейс для NAT |
| `--iface-name` | `awg0` | Имя интерфейса WireGuard |
| `--endpoint-v4` | автоопределение | IPv4 адрес эндпоинта (например, `1.2.3.4:51820`) |
| `--endpoint-v6` | автоопределение | IPv6 адрес эндпоинта (например, `[2001:db8::1]:51820`) |
| `--config` | `awg0.conf` | Путь к файлу конфигурации |

> **Примечание:** Флаги `--dns` и `--keepalive` принимаются, но не сохраняются в конфиг. DNS и keepalive жёстко заданы как `1.1.1.1, 8.8.8.8` и `25` соответственно при экспорте клиентских конфигов.

### client — Управление клиентами

#### client add — Добавление клиента

```bash
# С автоматическим назначением IP
amnezigo client add laptop

# С явным указанием IP
amnezigo client add phone --ipaddr 10.8.0.50

# С указанием файла конфигурации
amnezigo client add tablet --config /path/to/awg0.conf
```

Флаги:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--ipaddr` | авто | IP-адрес клиента (например, `10.8.0.5`) |
| `--config` | `awg0.conf` | Путь к файлу конфигурации сервера |

#### client list — Список клиентов

```bash
amnezigo client list
amnezigo client list --config /path/to/awg0.conf
```

Флаги:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--config` | `awg0.conf` | Путь к файлу конфигурации сервера |

#### client export — Экспорт конфигурации клиента

```bash
# Экспорт одного клиента (эндпоинт определяется автоматически)
amnezigo client export laptop

# Экспорт с конкретным протоколом обфускации
amnezigo client export laptop --protocol quic

# Экспорт всех клиентов
amnezigo client export
```

Флаги:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--protocol` | `random` | Протокол обфускации: `quic`, `dns`, `dtls`, `stun`, `random` |
| `--config` | `awg0.conf` | Путь к файлу конфигурации сервера |

> **Автоопределение эндпоинта:** Эндпоинт определяется автоматически в следующем порядке:
> 1. EndpointV4 из конфига сервера
> 2. EndpointV6 из конфига сервера
> 3. Внешний IP через icanhazip.com + порт сервера

#### client remove — Удаление клиента

```bash
amnezigo client remove laptop
amnezigo client remove phone --config /path/to/awg0.conf
```

Флаги:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--config` | `awg0.conf` | Путь к файлу конфигурации сервера |

### edge — Управление edge-серверами

Edge-серверы подключаются к хабу как WireGuard-клиенты (инициируют подключение). В отличие от обычных клиентов, в конфигах edge-серверов маршрутизация направлена только на IP хаба, и нет DNS.

#### edge add — Добавление edge-сервера

```bash
# С автоматическим назначением IP
amnezigo edge add edge1

# С явным указанием IP
amnezigo edge add edge2 --ipaddr 10.8.0.50
```

Флаги:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--ipaddr` | авто | IP-адрес edge-сервера (например, `10.8.0.5`) |
| `--config` | `awg0.conf` | Путь к файлу конфигурации сервера |

#### edge list — Список edge-серверов

```bash
amnezigo edge list
```

Флаги:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--config` | `awg0.conf` | Путь к файлу конфигурации сервера |

#### edge export — Экспорт конфигурации edge-сервера

```bash
# Экспорт одного edge-сервера
amnezigo edge export edge1

# Экспорт с конкретным протоколом обфускации
amnezigo edge export edge1 --protocol quic

# Экспорт всех edge-серверов
amnezigo edge export
```

Флаги:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--protocol` | `random` | Протокол обфускации: `quic`, `dns`, `dtls`, `stun`, `random` |
| `--config` | `awg0.conf` | Путь к файлу конфигурации сервера |

#### edge remove — Удаление edge-сервера

```bash
amnezigo edge remove edge1
```

Флаги:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--config` | `awg0.conf` | Путь к файлу конфигурации сервера |

### edit — Редактирование конфигурации сервера

```bash
# Разрешить трафик между клиентами
amnezigo edit --client-to-client true

# Запретить трафик между клиентами
amnezigo edit --client-to-client false
```

Флаги:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--client-to-client` | *(пусто)* | Разрешить/запретить трафик между клиентами (`true`/`false`) |
| `--config` | `awg0.conf` | Путь к файлу конфигурации сервера |

## Файлы конфигурации

### Конфиг сервера (awg0.conf)

```ini
[Interface]
PrivateKey = <приватный-ключ-сервера>
Address = 10.8.0.1/24
ListenPort = 55424
MTU = 1280
PostUp = iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE
PostDown = iptables -t nat -D POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE
Jc = 3
Jmin = 50
Jmax = 1000
S1 = 15
S2 = 16
S3 = 45
S4 = 10
H1 = 191091632-238083235
H2 = 469298095-484308427
H3 = 490129542-1366070158
H4 = 1959094164-1989726207
#_EndpointV4 = 1.2.3.4:51820
#_EndpointV6 = [2001:db8::1]:51820
#_ClientToClient = false
#_TunName = awg0

[Peer]
#_Role = client
#_Name = laptop
#_PrivateKey = <приватный-ключ-клиента>
PublicKey = <публичный-ключ-клиента>
AllowedIPs = 10.8.0.2/32

[Peer]
#_Role = edge
#_Name = edge1
#_PrivateKey = <приватный-ключ-edge>
PublicKey = <публичный-ключ-edge>
AllowedIPs = 10.8.0.3/32
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
S1 = 15
S2 = 16
S3 = 45
S4 = 10
H1 = 191091632-238083235
H2 = 469298095-484308427
H3 = 490129542-1366070158
H4 = 1959094164-1989726207
I1 = <b 0x16><b 0xfefd><b 0x0000><b 0x000000000000><b 0x0058>...
I2 = <b 0x16><b 0xfefd><b 0x0000><b 0x000000000000><b 0x0038>...
I3 = <b 0x16><b 0xfefd><b 0x0000><b 0x000000000000><b 0x0028>...
I4 = <b 0x16><b 0xfefd><b 0x0000><b 0x000000000000><b 0x0020>...
I5 = 

[Peer]
PublicKey = <публичный-ключ-сервера>
PresharedKey = <psk>
Endpoint = 1.2.3.4:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
```

### Конфиг edge-сервера (edge1.conf)

```ini
[Interface]
PrivateKey = <приватный-ключ-edge>
Address = 10.8.0.3/32
MTU = 1280
Jc = 3
Jmin = 50
Jmax = 1000
S1 = 15
S2 = 16
S3 = 45
S4 = 10
H1 = 191091632-238083235
H2 = 469298095-484308427
H3 = 490129542-1366070158
H4 = 1959094164-1989726207
I1 = <b 0x16><b 0xfefd><b 0x0000><b 0x000000000000><b 0x0058>...
I2 = <b 0x16><b 0xfefd><b 0x0000><b 0x000000000000><b 0x0038>...
I3 = <b 0x16><b 0xfefd><b 0x0000><b 0x000000000000><b 0x0028>...
I4 = <b 0x16><b 0xfefd><b 0x0000><b 0x000000000000><b 0x0020>...
I5 = 

[Peer]
PublicKey = <публичный-ключ-сервера>
PresharedKey = <psk>
Endpoint = 1.2.3.4:51820
AllowedIPs = 10.8.0.1/32
PersistentKeepalive = 25
```

Ключевые отличия от конфига клиента:
- **Нет строки DNS** (edge-серверам не нужен DNS)
- **AllowedIPs** указывает только на IP хаба (`10.8.0.1/32`), а не `0.0.0.0/0`

## Параметры обфускации

AmneziaWG использует несколько параметров для обфускации трафика WireGuard:

| Параметр | Описание |
|----------|----------|
| **Jc** | Количество мусорных пакетов перед реальными данными |
| **Jmin** | Минимальный размер мусорного пакета |
| **Jmax** | Максимальный размер мусорного пакета |
| **S1-S4** | Размерные префиксы для обфускации заголовков |
| **H1-H4** | Диапазоны значений заголовков (формат `min-max`) |
| **I1-I5** | Custom Packet Strings (CPS) — генерируются для каждого клиента при экспорте |

### Протоколы обфускации

Каждый протокол генерирует паттерны I1-I5, имитирующие реальный трафик:

| Протокол | Имитация |
|----------|----------|
| **quic** | QUIC Initial packet с длинным заголовком |
| **dns** | DNS Query с транзакционным ID и структурой запроса |
| **dtls** | DTLS 1.2 ClientHello с рукопожатием |
| **stun** | STUN Binding Request с magic cookie |
| **random** | Выбирает один из протоколов (детерминированно: `"random"` → DTLS) |

> **Примечание:** Протокол `random` не является действительно случайным — он детерминированно выбирает DTLS из-за `len("random") % 4 = 2`.

## Использование как библиотеки

Amnezigo можно использовать как Go-библиотеку для программного управления конфигурациями:

```go
import "github.com/Arsolitt/amnezigo"

func main() {
    // Создание менеджера
    mgr := amnezigo.NewManager("awg0.conf")
    
    // Добавление клиента
    peer, err := mgr.AddClient("laptop", "")
    if err != nil {
        log.Fatal(err)
    }
    
    // Добавление edge-сервера
    edge, err := mgr.AddEdge("edge1", "")
    if err != nil {
        log.Fatal(err)
    }
    
    // Экспорт конфигурации клиента
    clientCfg, err := mgr.ExportClient("laptop", "quic", "1.2.3.4:51820")
    if err != nil {
        log.Fatal(err)
    }
    
    // Экспорт конфигурации edge-сервера
    edgeCfg, err := mgr.ExportEdge("edge1", "quic", "1.2.3.4:51820")
    if err != nil {
        log.Fatal(err)
    }
}
```

Дополнительные примеры и документация API: [docs/library-usage.ru.md](docs/library-usage.ru.md)

## Лицензия

MIT License
