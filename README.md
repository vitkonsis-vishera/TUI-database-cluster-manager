Cluster TUI 🐘 status & management console

English | Русский

🇬🇧 English

Cluster TUI is a terminal user interface console written in Go, designed for monitoring and managing high-availability PostgreSQL clusters managed by Patroni.

Built on top of the modern Charm ecosystem (bubbletea, lipgloss, bubbles), it provides real-time visibility into cluster topology, replication lag, and node performance metrics.

🚀 Key Features

Auto-discovery: Connect to a single Patroni REST API endpoint and automatically discover all other cluster members.

Topology Monitoring: View node roles (Leader, Sync Standby, Replica), health state, Timeline numbers, and Replication Lag (in MB).

PostgreSQL Metrics: Directly queries nodes using pgx/v5 to track active connection usage (max_connections) and buffer pool efficiency (Cache Hit Ratio).

High Availability Client: Fallback mechanism that cycles through available Patroni endpoints if a node becomes unreachable.

Modern TUI Experience: Keyboard-driven tabbed layout with crisp color styling.

🛠 Tech Stack

Language: Go 1.25.0

TUI Frameworks:

charmbracelet/bubbletea — Elm-inspired TUI architecture

charmbracelet/lipgloss — Terminal styling and layout definition

charmbracelet/bubbles — TUI components (tables, text inputs)

Database Driver: jackc/pgx/v5 — PostgreSQL connection pooling

📋 Prerequisites

Go 1.25 or higher installed.

Access to Patroni REST API (default port 8008 or 8001).

Access to PostgreSQL instances (default port 5432) with read permissions for system stats (pg_stat_activity, pg_statio_user_tables).

📦 Installation & Run

Clone the repository:

git clone https://github.com/your-username/cluster-tui.git
cd cluster-tui


Download dependencies:

go mod download


Run directly:

go run main.go


Or build a binary:

go build -o cluster-tui main.go
./cluster-tui


⌨️ Keybindings & Navigation

Connection Tab (0: Connect)

Tab / ↑ / ↓ — Cycle between input fields (Patroni URL, PG User, PG Password).

Enter — Initiate cluster connection.

Main Views (Tabs 1-4)

1 .. 4 — Switch directly to tabs (1: Topology, 2: Metrics, 3: Actions, 4: Event Logs).

Tab / l / → — Next tab.

Shift+Tab / h / ← — Previous tab.

Ctrl+C or q — Quit application.

🗺 Roadmap

[x] Patroni REST API & PostgreSQL pool connection logic.

[x] Periodic cluster status polling (every 2s).

[x] Topology view with Lag and Timeline metrics.

[x] Connection stats and Cache Hit Ratio tracking.

[ ] Interactive Patroni cluster management from TUI:

[ ] Graceful Switchover / Force Failover

[ ] Reinitialize Replica

[ ] Pause / Resume maintenance mode

[ ] Real-time Patroni event stream viewer.

📄 License

MIT License

🇷🇺 Русский

Cluster TUI — это консольный терминальный интерфейс (TUI), написанный на Go, предназначенный для мониторинга и управления высокодоступными кластерами PostgreSQL под управлением Patroni.

Интерфейс построен на современной экосистеме Charm (bubbletea, lipgloss, bubbles) и позволяет отслеживать топологию, отставание репликации и метрики производительности базы данных в реальном времени.

🚀 Основные возможности

Автообнаружение нод (Auto-discovery): Подключение к одной точке входа Patroni REST API с последующим автоматическим обнаружением всех участников кластера.

Мониторинг топологии: Отображение текущих ролей (Leader, Sync Standby, Replica), статусов, номеров Timeline, а также отставания репликации (Replication Lag в MB).

Метрики PostgreSQL: Подключение к нодам напрямую через pgx/v5 для сбора количества активных соединений (max_connections) и эффективности кэша (Cache Hit Ratio).

Отказоустойчивость: Поддержка автоматического фолбэка при опросе клиентов Patroni в случае недоступности текущей ноды.

Современный TUI: Удобная навигация по вкладкам, понятные цветовые акценты и полная поддержка клавиатурного управления.

🛠 Технологический стек

Язык: Go 1.25.0

TUI Фреймворки:

charmbracelet/bubbletea — архитектура TUI (Elm Architecture)

charmbracelet/lipgloss — стилизация и макеты

charmbracelet/bubbles — компоненты (таблицы, поля ввода)

Драйвер БД: jackc/pgx/v5 — пул подключений к PostgreSQL

📋 Предварительные требования

Установленный Go версии 1.25 или выше.

Доступ к Patroni REST API (по умолчанию порт 8008 или 8001).

Доступ к PostgreSQL (по умолчанию порт 5432) с правами на чтение системных представлений (pg_stat_activity, pg_statio_user_tables).

📦 Установка и запуск

Клонируйте репозиторий:

git clone https://github.com/your-username/cluster-tui.git
cd cluster-tui


Загрузите зависимости:

go mod download


Соберите и запустите приложение:

go run main.go


Или скомпилируйте бинарный файл:

go build -o cluster-tui main.go
./cluster-tui


⌨️ Навигация и горячие клавиши

Вкладка подключения (0: Connect)

Tab / ↑ / ↓ — Переключение между полями ввода (Patroni URL, PG User, PG Password).

Enter — Подключиться к кластеру.

Основные вкладки (1-4)

1 .. 4 — Быстрый переход по вкладкам (1: Topology, 2: Metrics, 3: Actions, 4: Event Logs).

Tab / l / → — Следующая вкладка.

Shift+Tab / h / ← — Предыдущая вкладка.

Ctrl+C или q — Выход из приложения.

🗺 Дорожная карта (Roadmap)

[x] Авторизация и подключение к Patroni + PostgreSQL.

[x] Автоматический опрос статуса кластера (polling каждые 2 секунды).

[x] Таблица топологии с отображением Lag и Timeline.

[x] Мониторинг соединений и Cache Hit Ratio.

[ ] Интерактивные действия Patroni из TUI:

[ ] Graceful Switchover / Force Failover

[ ] Reinitialize Replica

[ ] Pause / Resume maintenance mode

[ ] Просмотр логов событий в режиме реального времени.

📄 Лицензия

MIT License
