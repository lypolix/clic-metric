# Clic-Metric — сервис коротких ссылок и статистики переходов

**Clic-Metric** — HTTP-сервис на Go для генерации коротких ссылок с хранением в PostgreSQL и подсчетом количества переходов по ним. Сервис clic-metric помогает бизнесу эффективно собирать и анализировать метрики посещений через короткие ссылки. Он упрощает создание удобных для пользователей сокращённых URL и фиксирует переходы по ним, позволяя отслеживать активность аудитории. 

---

## Возможности

- Создание коротких ссылок для любых URL (POST /url)
- Быстрый редирект по короткой ссылке (GET /{alias})
- Получение статистики переходов (GET /metrics/{alias})
- Basic Auth для эндпоинта создания ссылок
- Конфигурация через YAML и переменные окружения
- Логирование запросов и ошибок
- Тесты для основных функций

---

## Установка и запуск

### 1. Клонирование репозитория

git clone https://github.com/lypolix/clic-metric.git
cd clic-metric

### 2. Сборка

go build -o clic-metric ./cmd/clic-metric/main.go

### 3. Настройка базы данных

Запустите PostgreSQL и создайте базу и пользователя.

Пример строки подключения прописан в файле `config/local.yaml`:

storage_path: "host=localhost port=5432 user=postgres password=qazwsxedc dbname=postgres sslmode=disable"
http_server:
address: "localhost:8082"
timeout: 4s
idle_timeout: 60s
user: "myuser"
password: "mypass"

> (параметры согласно вашей среде.)

### 4. Настройка переменной окружения для конфига

export CONFIG_PATH=./config/local.yaml

### 5. Запуск сервиса

./clic-metric
