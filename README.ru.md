# Amnezigo

[![Go Reference](https://pkg.go.dev/badge/github.com/Arsolitt/amnezigo.svg)](https://pkg.go.dev/github.com/Arsolitt/amnezigo)
[![Go Report Card](https://goreportcard.com/badge/github.com/Arsolitt/amnezigo)](https://goreportcard.com/report/github.com/Arsolitt/amnezigo)

**Amnezia** + **Go** = **Amnezigo**

CLI-утилита и Go-библиотека для генерации и управления конфигурациями [AmneziaWG](https://github.com/amnezia-vpn/amneziawg) v2.0.

## Возможности

- Генерация серверных конфигураций AmneziaWG с параметрами обфускации
- Управление пирами: добавление, удаление, просмотр списка, экспорт
- Несколько протоколов обфускации (QUIC, DNS, DTLS, STUN)
- Автоматическое назначение IP-адресов для пиров
- Генерация правил iptables для NAT и проброса трафика
- Параметры обфускации для каждого пира при экспорте
- Динамическое переключение режима client-to-client
- Автоопределение эндпоинта (IPv4/IPv6)
- Использование как Go-библиотеки

## Установка

### go install

```bash
go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest
```

### Сборка из исходников

```bash
git clone https://github.com/Arsolitt/amnezigo.git
cd amnezigo
go build -o amnezigo ./cmd/amnezigo/
```

### Docker

```bash
docker build -t amnezigo .
docker run --rm -v $(pwd):/data amnezigo init --ipaddr 10.8.0.1/24
```

## Быстрый старт

```bash
# Инициализация серверной конфигурации
amnezigo init --ipaddr 10.8.0.1/24

# Добавление пира
amnezigo add laptop

# Экспорт конфигурации пира
amnezigo export laptop
```

## Использование

### Инициализация сервера

```bash
amnezigo init --ipaddr 10.8.0.1/24
```

Параметры:
- `--ipaddr` — IP-адрес сервера с подсетью (обязательно)
- `--port` — Порт для прослушивания (по умолчанию: случайный 10000-65535)
- `--mtu` — Размер MTU (по умолчанию: 1280)
- `--dns` — DNS-серверы (по умолчанию: "1.1.1.1, 8.8.8.8") — не сохраняется в конфиг
- `--keepalive` — Интервал keepalive (по умолчанию: 25) — не сохраняется в конфиг
- `--client-to-client` — Разрешить трафик между клиентами
- `--iface` — Основной сетевой интерфейс для NAT (по умолчанию: автоопределение)
- `--iface-name` — Имя интерфейса WireGuard (по умолчанию: awg0)
- `--endpoint-v4` — IPv4 адрес эндпоинта (автоопределение если пусто)
- `--endpoint-v6` — IPv6 адрес эндпоинта (автоопределение если пусто)
- `--config` — Путь к файлу конфигурации (по умолчанию: awg0.conf)

Примечание: Флаги `--dns` и `--keepalive` принимаются для совместимости, но не сохраняются в серверной конфигурации. DNS жёстко задан как "1.1.1.1, 8.8.8.8", а keepalive как 25 в экспортируемых конфигах.

### Команды для работы с пирами

#### Добавление пира

```bash
# Автоматическое назначение IP
amnezigo add laptop

# Явное указание IP
amnezigo add phone --ipaddr 10.8.0.50
```

Параметры:
- `--ipaddr` — IP-адрес пира (автоназначение если не указан)
- `--config` — Файл конфигурации сервера (по умолчанию: awg0.conf)

#### Список пиров

```bash
amnezigo list
```

Параметры:
- `--config` — Файл конфигурации сервера (по умолчанию: awg0.conf)

#### Экспорт конфигурации пира

```bash
# Экспорт одного пира
amnezigo export laptop

# Экспорт с конкретным протоколом
amnezigo export laptop --protocol quic

# Экспорт всех пиров
amnezigo export
```

Параметры:
- `--protocol` — Протокол обфускации: random, quic, dns, dtls, stun (по умолчанию: random)
- `--config` — Файл конфигурации сервера (по умолчанию: awg0.conf)

Эндпоинт определяется автоматически в следующем порядке:
1. Сохранённый IPv4 эндпоинт из конфига сервера (`EndpointV4`)
2. Сохранённый IPv6 эндпоинт из конфига сервера (`EndpointV6`)
3. Автоопределение через HTTP-сервис icanhazip.com

#### Удаление пира

```bash
amnezigo remove laptop
```

Параметры:
- `--config` — Файл конфигурации сервера (по умолчанию: awg0.conf)

### Редактирование конфигурации сервера

```bash
# Разрешить трафик между клиентами
amnezigo edit --client-to-client true

# Запретить трафик между клиентами
amnezigo edit --client-to-client false
```

Параметры:
- `--client-to-client` — Включить/выключить client-to-client (true/false)
- `--config` — Файл конфигурации сервера (по умолчанию: awg0.conf)

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
H1 = 191091632-238083235
H2 = 469298095-484308427
H3 = 490129542-1366070158
H4 = 1959094164-1989726207
#_EndpointV4 = 1.2.3.4:51820
#_EndpointV6 = [2001:db8::1]:51820
#_ClientToClient = false
#_TunName = awg0

[Peer]
#_Name = laptop
#_PrivateKey = <приватный-ключ-пира>
PublicKey = <публичный-ключ-пира>
AllowedIPs = 10.8.0.2/32
```

### Конфиг пира (laptop.conf)

```ini
[Interface]
PrivateKey = <приватный-ключ-пира>
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
I1 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0040><b 0x00><b 0x01><t><r 40>
I2 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0020><b 0x01><t><r 20>
I3 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0010><b 0x01><t><r 16>
I4 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0005><b 0x01><t><r 5>
I5 = 

[Peer]
PublicKey = <публичный-ключ-сервера>
PresharedKey = <psk>
Endpoint = 1.2.3.4:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
```

## Параметры обфускации

AmneziaWG использует несколько параметров для обфускации трафика WireGuard:

- **Jc, Jmin, Jmax** — Параметры мусорных пакетов
- **S1-S4** — Размерные префиксы
- **H1-H4** — Диапазоны значений заголовков (хранятся в формате min-max)
- **I1-I5** — Custom Packet Strings (CPS), генерируются для каждого пира при экспорте

### Синтаксис тегов CPS

I1-I5 используют теговый синтаксис для построения последовательностей байт:

| Тег | Описание | Пример |
|-----|----------|--------|
| `<b 0x...>` | Фиксированные байты в hex | `<b 0xc0ff>` |
| `<r N>` | Случайные байты (N байт) | `<r 8>` |
| `<t>` | Метка времени (4 байта) | `<t>` |
| `<c>` | Счётчик | `<c>` |
| `<rc N>` | Случайные символы (N байт) | `<rc 7>` |
| `<rd N>` | Случайные цифры (N байт) | `<rd 2>` |

### Протоколы

Каждый протокол генерирует различные паттерны I1-I5:

- **quic** — Имитирует QUIC Initial пакеты с длинными заголовками, DCID, метками времени
- **dns** — Имитирует DNS query пакеты с транзакционными ID и структурой домена
- **dtls** — Имитирует DTLS 1.2 ClientHello пакеты с заголовками рукопожатия
- **stun** — Имитирует STUN Binding Request пакеты с magic cookie
- **random** — Выбирает протокол детерминированно на основе длины строки (по умолчанию DTLS)

## Использование как библиотеки

Amnezigo можно использовать как Go-библиотеку. Подробная документация: [docs/library-usage.md](docs/library-usage.md).

```go
import "github.com/Arsolitt/amnezigo"

func main() {
    // Генерация пары ключей
    privateKey, publicKey := amnezigo.GenerateKeyPair()
    
    // Создание менеджера для операций с конфигурацией
    mgr := amnezigo.NewManager("awg0.conf")
    
    // Добавление пира
    peer, err := mgr.AddPeer("laptop", "")
    if err != nil {
        log.Fatal(err)
    }
    
    // Экспорт конфигурации пира
    clientCfg, err := mgr.ExportPeer("laptop", "quic", "1.2.3.4:51820")
    if err != nil {
        log.Fatal(err)
    }
}
```

## Использование с ИИ-ассистентами

Рекомендуется скопировать следующий промпт и отправить его ИИ-ассистенту — это может значительно улучшить качество генерируемых конфигураций AmneziaWG:

```
https://raw.githubusercontent.com/Arsolitt/amnezigo/refs/heads/main/docs/llms-full.txt This link is the full documentation of Amnezigo.

【Role Setting】
You are an expert proficient in network protocols and AmneziaWG configuration.

【Task Requirements】
1. Knowledge Base: Please read and deeply understand the content of this link, and use it as the sole basis for answering questions and writing configurations.
2. No Hallucinations: Absolutely do not fabricate fields that do not exist in the documentation. If the documentation does not mention it, please tell me directly "Documentation does not mention".
3. Default Format: Output INI format configuration by default (unless I explicitly request a different format), and add key comments.
4. Exception Handling: If you cannot access this link, please inform me clearly and prompt me to manually download the documentation and upload it to you.
```

## Лицензия

[MIT](LICENSE)
