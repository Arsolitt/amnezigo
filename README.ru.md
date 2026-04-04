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

## Быстрый старт

```bash
# Установка
go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest

# Инициализация сервера
amnezigo init --ipaddr 10.8.0.1/24

# Добавление пира
amnezigo add laptop

# Экспорт конфигурации пира
amnezigo export laptop
```

## Документация

| Страница | Описание |
|----------|----------|
| [Установка и быстрый старт](docs/installation.md) | Способы установки, Docker, пошаговое руководство |
| [Справочник CLI](docs/cli-reference.md) | Все команды, флаги, значения по умолчанию |
| [Файлы конфигурации](docs/configuration.md) | Формат конфигов сервера и клиента, метаданные, iptables |
| [Использование как библиотеки](docs/library-usage.md) | Программный API: Manager, ввод/вывод, ключи, обфускация |
| [Параметры обфускации](docs/obfuscation.md) | Мусорные пакеты, размерные префиксы, заголовки, CPS-теги, протоколы |

> **Примечание:** Документация доступна на английском языке.

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
