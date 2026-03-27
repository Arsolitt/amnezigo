# Использование amnezigo как Go-библиотеки

Проект `amnezigo` (`github.com/Arsolitt/amnezigo`) — это одновременно CLI-утилита и Go-библиотека для управления конфигурациями AmneziaWG v2.0. Корневой пакет `amnezigo` экспортирует всю бизнес-логику, а CLI-команды в `internal/cli/` являются тонкими обёртками над библиотекой.

## Установка

```bash
go get github.com/Arsolitt/amnezigo
```

```go
import "github.com/Arsolitt/amnezigo"
```

## Быстрый старт

```go
package main

import (
    "fmt"
    "os"

    "github.com/Arsolitt/amnezigo"
)

func main() {
    // Создаём менеджер для работы с конфигом сервера
    manager := amnezigo.NewManager("/etc/amnezia/awg0.conf")

    // Загружаем существующий конфиг
    cfg, err := manager.Load()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Ошибка загрузки: %v\n", err)
        os.Exit(1)
    }

    // Добавляем нового клиента (IP назначается автоматически)
    peer, err := manager.AddClient("laptop", "")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Ошибка добавления: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("Добавлен клиент: %s (IP: %s)\n", peer.Name, peer.AllowedIPs)

    // Экспортируем конфигурацию клиента для передачи ему
    clientCfg, err := manager.ExportClient("laptop", "quic", "203.0.113.50:51820")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Ошибка экспорта: %v\n", err)
        os.Exit(1)
    }

    // Записываем конфиг клиента в файл
    file, err := os.Create("laptop.conf")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    if err := amnezigo.WriteClientConfig(file, clientCfg); err != nil {
        panic(err)
    }
    fmt.Println("Конфиг клиента сохранён в laptop.conf")
}
```

## API Manager

`Manager` — высокоуровневый интерфейс для управления конфигурацией сервера WireGuard/AmneziaWG, клиентами и edge-серверами.

### Создание менеджера

```go
manager := amnezigo.NewManager("/etc/amnezia/awg0.conf")
```

### Загрузка и сохранение конфигурации

```go
// Загрузка конфигурации с диска
cfg, err := manager.Load()
if err != nil {
    panic(err)
}

// Сохранение конфигурации (атомарная запись)
err = manager.Save(cfg)
if err != nil {
    panic(err)
}
```

### Управление клиентами

#### Добавление клиента

```go
// Добавление с автоматическим назначением IP
peer, err := manager.AddClient("phone", "")
if err != nil {
    panic(err)
}

// Добавление с явным указанием IP
peer, err = manager.AddClient("tablet", "10.8.0.100")
if err != nil {
    panic(err)
}
```

**Важно:** Имя должно быть уникальным среди всех пиров (клиентов и edge-серверов).

#### Удаление клиента

```go
err := manager.RemoveClient("phone")
if err != nil {
    panic(err)
}
```

#### Поиск клиента

```go
peer, err := manager.FindClient("laptop")
if err != nil {
    fmt.Printf("Клиент не найден: %v\n", err)
    return
}
fmt.Printf("Найден клиент: %s\n", peer.Name)
```

#### Список всех клиентов

```go
peers := manager.ListClients()
for _, peer := range peers {
    fmt.Printf("- %s: %s\n", peer.Name, peer.AllowedIPs)
}
```

#### Экспорт конфигурации клиента

```go
// Экспорт по имени клиента
clientCfg, err := manager.ExportClient("laptop", "quic", "203.0.113.50:51820")
if err != nil {
    panic(err)
}

// Или построение конфигурации из известного PeerConfig
peer, _ := manager.FindClient("laptop")
clientCfg, err := manager.BuildClientConfig(peer, "dns", "203.0.113.50:51820")
```

**Параметры протокола:**
- `"quic"` — имитация QUIC Initial пакетов
- `"dns"` — имитация DNS пакетов
- `"dtls"` — имитация DTLS пакетов
- `"stun"` — имитация STUN пакетов
- `"random"` — детерминированный выбор шаблона (на основе длины строки)

### Управление edge-серверами

Edge-серверы подключаются к хабу как WireGuard-клиенты (инициируют подключение). Они делят общий IP-пул с обычными клиентами.

#### Добавление edge-сервера

```go
// Добавление с автоматическим назначением IP
edge, err := manager.AddEdge("edge1", "")
if err != nil {
    panic(err)
}

// Добавление с явным указанием IP
edge, err = manager.AddEdge("edge2", "10.8.0.100")
if err != nil {
    panic(err)
}
```

**Важно:** Имя должно быть уникальным среди всех пиров (клиентов и edge-серверов).

#### Удаление edge-сервера

```go
err := manager.RemoveEdge("edge1")
if err != nil {
    panic(err)
}
```

#### Поиск edge-сервера

```go
edge, err := manager.FindEdge("edge1")
if err != nil {
    fmt.Printf("Edge-сервер не найден: %v\n", err)
    return
}
fmt.Printf("Найден edge-сервер: %s\n", edge.Name)
```

#### Список всех edge-серверов

```go
edges := manager.ListEdges()
for _, edge := range edges {
    fmt.Printf("- %s: %s\n", edge.Name, edge.AllowedIPs)
}
```

#### Экспорт конфигурации edge-сервера

В отличие от экспорта клиента, конфиг edge-сервера не содержит DNS и маршрутизирует только на IP хаба.

```go
// Возвращает сериализованный конфиг как []byte
edgeData, err := manager.ExportEdge("edge1", "quic", "203.0.113.50:51820")
if err != nil {
    panic(err)
}

// Запись в файл с ограниченными правами
if err := os.WriteFile("edge1.conf", edgeData, 0600); err != nil {
    log.Fatal(err)
}
```

#### Построение конфигурации edge-сервера

```go
// Возвращает ClientConfig (edge переиспользует тип ClientConfig)
edgeCfg, err := manager.BuildEdgeConfig("edge1", "quic", "203.0.113.50:51820")
if err != nil {
    panic(err)
}
```

**Важно:** `BuildEdgeConfig` принимает `name` (строку, а не `PeerConfig`) и ищет edge-сервер внутри.

## Парсинг и запись конфигураций

### Парсинг конфигурации сервера

```go
// Из файла
cfg, err := amnezigo.LoadServerConfig("/etc/amnezia/awg0.conf")

// Из io.Reader
file, _ := os.Open("awg0.conf")
defer file.Close()
cfg, err := amnezigo.ParseServerConfig(file)
```

### Запись конфигурации сервера

```go
// В файл (атомарная запись через .tmp)
err := amnezigo.SaveServerConfig("/etc/amnezia/awg0.conf", cfg)

// В io.Writer
var buf bytes.Buffer
err := amnezigo.WriteServerConfig(&buf, cfg)
```

### Запись конфигурации клиента/edge

```go
// В io.Writer
file, _ := os.Create("client.conf")
defer file.Close()
err := amnezigo.WriteClientConfig(file, clientCfg)
```

## Генерация ключей

### Генерация пары ключей

```go
privateKey, publicKey := amnezigo.GenerateKeyPair()
fmt.Printf("Private: %s\n", privateKey) // 44 символа, base64
fmt.Printf("Public:  %s\n", publicKey)  // 44 символа, base64
```

**Важно:** Функция паникует при ошибке `crypto/rand`, так как это считается неисправимой системной ошибкой.

### Вывод публичного ключа из приватного

```go
publicKey := amnezigo.DerivePublicKey(privateKey)
```

**Важно:** Функция паникует при невалидном base64 или неправильной длине ключа.

### Генерация PresharedKey

```go
psk := amnezigo.GeneratePSK() // 44 символа, base64
```

## Обфускация

### Генерация клиентской конфигурации обфускации

```go
// Генерирует полную ClientObfuscationConfig с I1-I5
clientObf := amnezigo.GenerateConfig("quic", 1280, 15, 3)
```

**Параметры:**
- `protocol` — протокол обфускации (`"quic"`, `"dns"`, `"dtls"`, `"stun"`, `"random"`)
- `mtu` — MTU интерфейса
- `s1` — размер префикса S1
- `jc` — параметр мусора Jc

**Особенность:** `GenerateConfig` использует точечные диапазоны H1-H4 (Min == Max).

### Генерация серверной конфигурации обфускации

```go
// Генерирует ServerObfuscationConfig без I1-I5
serverObf := amnezigo.GenerateServerConfig(0, 15, 3)
```

**Важно:** Параметр протокола (первый аргумент) игнорируется — сервер не использует CPS-строки.

**Особенность:** `GenerateServerConfig` использует истинные диапазоны H1-H4 (Min < Max).

### Генерация CPS-строк

```go
i1, i2, i3, i4, i5 := amnezigo.GenerateCPS("quic", 1280, 15, 0)
fmt.Printf("I1: %s\n", i1) // например: "<b 0xc0ff00000001>..."
```

**Важно:** Четвёртый параметр не используется.

### Генерация отдельных компонентов

```go
// Заголовки H1-H4 (непересекающиеся, из разных регионов uint32)
headers := amnezigo.GenerateHeaders()
fmt.Printf("H1: %d, H2: %d, H3: %d, H4: %d\n", 
    headers.H1, headers.H2, headers.H3, headers.H4)

// Префиксы размеров S1-S4
prefixes := amnezigo.GenerateSPrefixes()
fmt.Printf("S1: %d, S2: %d, S3: %d, S4: %d\n",
    prefixes.S1, prefixes.S2, prefixes.S3, prefixes.S4)

// Параметры мусора Jc, Jmin, Jmax
junk := amnezigo.GenerateJunkParams()
fmt.Printf("Jc: %d, Jmin: %d, Jmax: %d\n", junk.Jc, junk.Jmin, junk.Jmax)

// Диапазоны заголовков H1-H4 (непересекающиеся)
ranges := amnezigo.GenerateHeaderRanges()
fmt.Printf("H1: %d-%d\n", ranges[0].Min, ranges[0].Max)
```

## CPS-конструкция

CPS (Custom Packet String) — это механизм построения кастомных пакетов обфускации.

### Создание тегов

```go
// Байты в hex-формате
tag := amnezigo.BuildCPSTag("b", "0xc0ff") // "<b 0xc0ff>"
tag = amnezigo.BuildCPSTag("b", "c0ff")    // "<b 0xc0ff>" (0x добавляется автоматически)

// Случайные байты
tag = amnezigo.BuildCPSTag("r", "16")  // "<r 16>" — 16 случайных байт

// Случайные ASCII символы
tag = amnezigo.BuildCPSTag("rc", "8") // "<rc 8>" — 8 случайных ASCII символов

// Случайные цифры
tag = amnezigo.BuildCPSTag("rd", "4") // "<rd 4>" — 4 случайные цифры

// Счётчик
tag = amnezigo.BuildCPSTag("c", "") // "<c>"

// Timestamp
tag = amnezigo.BuildCPSTag("t", "") // "<t>"
```

### Объединение тегов в CPS

```go
cps := amnezigo.BuildCPS([]string{
    "<b 0xc0ff00000001>",
    "<r 8>",
    "<t>",
    "<r 40>",
})
// Результат: "<b 0xc0ff00000001><r 8><t><r 40>"
```

### Полный пример построения CPS

```go
tags := []string{
    amnezigo.BuildCPSTag("b", "0xc0ff"),
    amnezigo.BuildCPSTag("b", "00000001"),
    amnezigo.BuildCPSTag("b", "08"),
    amnezigo.BuildCPSTag("r", "8"),
    amnezigo.BuildCPSTag("t", ""),
    amnezigo.BuildCPSTag("r", "40"),
}
cps := amnezigo.BuildCPS(tags)
```

## Шаблоны протоколов

### Получение шаблона

```go
// QUIC — имитация QUIC Initial пакетов
quicTmpl := amnezigo.QUICTemplate()

// DNS — имитация DNS пакетов
dnsTmpl := amnezigo.DNSTemplate()

// DTLS — имитация DTLS пакетов
dtlsTmpl := amnezigo.DTLSTemplate()

// STUN — имитация STUN пакетов
stunTmpl := amnezigo.STUNTemplate()
```

### Структура шаблона

```go
type I1I5Template struct {
    I1, I2, I3, I4, I5 []TagSpec
}

type TagSpec struct {
    Type  string // "bytes", "random", "random_chars", "random_digits", "counter", "timestamp"
    Value string // значение зависит от типа
}
```

### Пример использования шаблона

```go
tmpl := amnezigo.QUICTemplate()
for _, tag := range tmpl.I1 {
    fmt.Printf("Tag: %s = %s\n", tag.Type, tag.Value)
}
```

## Сетевые утилиты

### Валидация IP-адреса

```go
valid := amnezigo.IsValidIPAddr("10.8.0.1/24") // true
valid = amnezigo.IsValidIPAddr("10.8.0.1")      // false (требуется CIDR)
valid = amnezigo.IsValidIPAddr("invalid")       // false
```

### Извлечение подсети

```go
subnet := amnezigo.ExtractSubnet("10.8.0.100/24") // "10.8.0.0/24"
subnet = amnezigo.ExtractSubnet("invalid")        // "invalid" (возвращает как есть)
```

### Генерация случайного порта

```go
port, err := amnezigo.GenerateRandomPort()
if err != nil {
    panic(err)
}
fmt.Printf("Порт: %d\n", port) // диапазон [10000, 65535]
```

### Определение основного интерфейса

```go
iface := amnezigo.DetectMainInterface()
fmt.Printf("Основной интерфейс: %s\n", iface) // например: "eth0"
```

**Примечание:** Возвращает первый non-loopback интерфейс, который UP и имеет адреса.

### Поиск следующего доступного IP

```go
existingIPs := []string{"10.8.0.2", "10.8.0.3", "10.8.0.10"}
ip, err := amnezigo.FindNextAvailableIP("10.8.0.1/24", existingIPs)
if err != nil {
    panic(err)
}
fmt.Printf("Следующий IP: %s\n", ip) // "10.8.0.4"
```

**Алгоритм:** Перебирает адреса от .2 до .254, пропуская занятые.

## Правила iptables

### Генерация PostUp

```go
postUp := amnezigo.GeneratePostUp("awg0", "eth0", "10.8.0.0/24", false)
// Возвращает строку с правилами, разделёнными "; "
```

**Правила:**
1. ACCEPT INPUT/OUTPUT на туннельном интерфейсе
2. FORWARD с туннеля на основной интерфейс
3. ACCEPT established/related соединений
4. MASQUERADE для NAT

### Генерация PostDown

```go
postDown := amnezigo.GeneratePostDown("awg0", "eth0", "10.8.0.0/24", false)
// Те же правила, но с -D (delete) вместо -A (append)
```

### С client-to-client трафиком

```go
// Включает правило для пересылки между клиентами
postUp := amnezigo.GeneratePostUp("awg0", "eth0", "10.8.0.0/24", true)
```

## Справочник типов

### Ролевые константы

```go
const (
    RoleClient = "client"
    RoleEdge   = "edge"
)
```

### ServerConfig

```go
type ServerConfig struct {
    Clients     []PeerConfig
    Edges       []PeerConfig
    Interface   InterfaceConfig
    Obfuscation ServerObfuscationConfig
}
```

### InterfaceConfig

```go
type InterfaceConfig struct {
    PrivateKey     string
    PublicKey      string
    Address        string    // CIDR, например "10.8.0.1/24"
    PostUp         string
    PostDown       string
    MainIface      string    // Основной интерфейс (eth0, ens18, etc.)
    TunName        string    // Имя туннеля (awg0)
    EndpointV4     string
    EndpointV6     string
    ListenPort     int
    MTU            int
    ClientToClient bool
}
```

### PeerConfig

```go
type PeerConfig struct {
    CreatedAt         time.Time
    ClientObfuscation *ClientObfuscationConfig
    Name              string
    Role              string  // "client" или "edge"
    PrivateKey        string
    PublicKey         string
    PresharedKey      string
    AllowedIPs        string // CIDR пира, например "10.8.0.2/32"
}
```

### ServerObfuscationConfig

```go
type ServerObfuscationConfig struct {
    Jc, Jmin, Jmax int
    S1, S2, S3, S4 int
    H1, H2, H3, H4 HeaderRange
}
```

### ClientObfuscationConfig

```go
type ClientObfuscationConfig struct {
    I1, I2, I3, I4, I5 string  // CPS-строки
    ServerObfuscationConfig    // встраивает серверные параметры
}
```

### ClientConfig

```go
type ClientConfig struct {
    Peer      ClientPeerConfig
    Interface ClientInterfaceConfig
}
```

### ClientInterfaceConfig

```go
type ClientInterfaceConfig struct {
    PrivateKey  string
    Address     string
    DNS         string  // Пусто для edge-серверов
    Obfuscation ClientObfuscationConfig
    MTU         int
}
```

### ClientPeerConfig

```go
type ClientPeerConfig struct {
    PublicKey           string
    PresharedKey        string
    Endpoint            string
    AllowedIPs          string  // "0.0.0.0/0, ::/0" для клиентов, "<hub_ip>/32" для edge
    PersistentKeepalive int
}
```

### Вспомогательные типы

```go
type HeaderRange struct {
    Min, Max uint32
}

type Headers struct {
    H1, H2, H3, H4 uint32
}

type SPrefixes struct {
    S1, S2, S3, S4 int
}

type JunkParams struct {
    Jc, Jmin, Jmax int
}

type CPSConfig struct {
    I1, I2, I3, I4, I5 string
}

type TagSpec struct {
    Type  string
    Value string
}

type I1I5Template struct {
    I1, I2, I3, I4, I5 []TagSpec
}

type Manager struct {
    ConfigPath string
}
```

## Особенности и ограничения

### Захардкоженные значения

| Параметр | Значение | Где используется |
|----------|----------|------------------|
| DNS | `"1.1.1.1, 8.8.8.8"` | `BuildClientConfig` (клиенты) |
| DNS | `""` (пусто) | `BuildEdgeConfig` (edge-серверы) |
| AllowedIPs | `"0.0.0.0/0, ::/0"` | `BuildClientConfig` (экспорт клиента) |
| AllowedIPs | `"<hub_ip>/32"` | `BuildEdgeConfig` (экспорт edge) |
| PersistentKeepalive | `25` | `BuildClientConfig` и `BuildEdgeConfig` |

### Поведение GenerateKeyPair и GeneratePSK

Эти функции паникуют при ошибке `crypto/rand.Read()`, так как это считается неисправимой системной ошибкой. В нормальных условиях это не должно происходить.

`DerivePublicKey()` также паникует при невалидном base64 или неправильной длине ключа.

### Различия GenerateConfig и GenerateServerConfig

| Функция | H1-H4 | Назначение |
|---------|-------|------------|
| `GenerateConfig` | Точечные (Min == Max) | Клиентская обфускация |
| `GenerateServerConfig` | Диапазоны (Min < Max) | Серверная обфускация |

### Уникальность имён пиров

Имена пиров (клиентов и edge-серверов) должны быть глобально уникальными. Клиент и edge-сервер не могут иметь одинаковое имя. Проверка `isNameTaken` охватывает оба среза (`Clients` и `Edges`).

### Общий IP-пул

Клиенты и edge-серверы делят один IP-пул. Функция `resolveClientIP` учитывает оба среза при поиске следующего доступного IP, чтобы избежать конфликтов.

### Протокол "random"

При указании протокола `"random"` выбор шаблона детерминирован и основан на длине строки протокола:
- `len("random") % 4` определяет выбор между QUIC, DNS, DTLS, STUN

### Игнорирование параметров

- `GenerateServerConfig(_, s1, jc)` игнорирует первый параметр (протокол)
- `GenerateCPS(protocol, mtu, s1, _)` игнорирует четвёртый параметр

### Edge-серверы: особенности

- `ExportEdge` возвращает `([]byte, error)`, в отличие от `ExportClient`, который возвращает `(ClientConfig, error)`
- `BuildEdgeConfig` принимает имя (строку), а не `PeerConfig`
- Edge-конфиги не содержат PostUp/PostDown — edge-серверы являются конечными точками трафика, а не маршрутизаторами
- Файлы конфигурации edge-серверов рекомендуется создавать с правами 0600

## Полный пример: создание сервера с нуля

```go
package main

import (
    "fmt"
    "os"

    "github.com/Arsolitt/amnezigo"
)

func main() {
    // Генерируем ключи сервера
    serverPriv, serverPub := amnezigo.GenerateKeyPair()

    // Генерируем параметры обфускации
    obf := amnezigo.GenerateServerConfig(0, 15, 3)

    // Определяем основной интерфейс
    mainIface := amnezigo.DetectMainInterface()
    if mainIface == "" {
        mainIface = "eth0"
    }

    // Генерируем случайный порт
    port, _ := amnezigo.GenerateRandomPort()

    // Создаём конфигурацию сервера
    cfg := amnezigo.ServerConfig{
        Interface: amnezigo.InterfaceConfig{
            PrivateKey: serverPriv,
            PublicKey:  serverPub,
            Address:    "10.8.0.1/24",
            ListenPort: port,
            MTU:        1280,
            TunName:    "awg0",
            MainIface:  mainIface,
            PostUp:     amnezigo.GeneratePostUp("awg0", mainIface, "10.8.0.0/24", false),
            PostDown:   amnezigo.GeneratePostDown("awg0", mainIface, "10.8.0.0/24", false),
        },
        Obfuscation: obf,
    }

    // Создаём менеджер и сохраняем конфиг
    manager := amnezigo.NewManager("/etc/amnezia/awg0.conf")
    if err := manager.Save(cfg); err != nil {
        fmt.Fprintf(os.Stderr, "Ошибка сохранения: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("Конфигурация сервера создана!")

    // Добавляем клиентов
    clients := []string{"laptop", "phone", "tablet"}
    for _, name := range clients {
        peer, err := manager.AddClient(name, "")
        if err != nil {
            fmt.Fprintf(os.Stderr, "Ошибка добавления %s: %v\n", name, err)
            continue
        }
        fmt.Printf("Добавлен клиент: %s -> %s\n", name, peer.AllowedIPs)

        // Экспортируем конфиг клиента
        clientCfg, err := manager.ExportClient(name, "quic", "203.0.113.50:51820")
        if err != nil {
            fmt.Fprintf(os.Stderr, "Ошибка экспорта %s: %v\n", name, err)
            continue
        }

        // Сохраняем в файл
        file, _ := os.Create(name + ".conf")
        amnezigo.WriteClientConfig(file, clientCfg)
        file.Close()
        fmt.Printf("  -> Конфиг сохранён в %s.conf\n", name)
    }
}
```
