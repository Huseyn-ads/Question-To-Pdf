# HTTP и безопасность

## Среда и cookies

MVP использует один origin: frontend и API на одном HTTPS-домене,
frontend обращается к `/v1/...`. Локально используйте dev proxy `/v1`
на backend и задайте `PUBLIC_ORIGIN` равным origin страницы frontend.
По умолчанию это `http://localhost:8080`. `localhost` и `127.0.0.1`
считаются разными origins. Не определяйте доверенный origin по Host,
X-Forwarded-Host или X-Forwarded-Proto из запроса.

В production нужны `APP_ENV=production` и явный
`PUBLIC_ORIGIN=https://ваш-домен`. При HTTP-origin production не запустится.
HTTPS завершается на reverse proxy; прямой публичный доступ к HTTP-порту
backend должен быть закрыт. Proxy должен сам ограничивать запросы и соединения.

Cookie `session`: `Path=/`, без Domain, HttpOnly, SameSite=Lax,
Secure для HTTPS-origin. Серверная сессия и Expires cookie действуют 7 суток.
Сервер принимает только неистёкшие сессии. В PostgreSQL хранится SHA-256
session token; исходный token существует только в cookie клиента.
Пароли хешируются bcrypt. Минимум — 8 Unicode-кодовых точек, максимум —
72 байта UTF-8. Email очищается от пробелов по краям и приводится к нижнему регистру.

## CSRF-контракт

1. `POST /v1/auth/login` с JSON email/password создаёт сессию и cookie.
   `POST /v1/auth/register` создаёт пользователя (201), затем нужен login.
   Для обоих запросов сервер требует точный `Origin: PUBLIC_ORIGIN`.
   Если Origin отсутствует, проверяется origin из Referer. Отсутствие обоих,
   `Origin: null`, посторонний origin или Sec-Fetch-Site=cross-site дают 403.
2. После login вызвать `GET /v1/auth/csrf` с cookie. Ответ:
   `{"csrf_token":"..."}`. Без сессии — 401. Ответы не кешируются.
3. Все POST/PUT/PATCH/DELETE защищённых маршрутов, включая multipart preview
   и logout, требуют заголовок `X-CSRF-Token` и доверенный Origin/Referer.
   Токен сравнивается с отдельным случайным csrf_token этой сессии,
   а не с session token. Токен другой сессии, пустой или неправильный — 403.
4. Хранить csrf_token в памяти frontend; после reload получать его заново.
   После каждого login получать новый token; после logout удалять token из памяти.
   При 401 сначала login, затем получение token; не повторять изменяющий запрос
   автоматически. При logout нужны действующая сессия и CSRF token;
   сервер удаляет запись сессии и очищает cookie с теми же именем и Path,
   возвращает 204 без тела. Повторный logout без сессии возвращает 401.

Для fetch достаточно `credentials: 'same-origin'` (значение по умолчанию).
Для Postman/curl тоже передавайте явный доверенный Origin, cookie и CSRF token.
Выполните новую миграцию `20261102120000_add_session_csrf_token.sql`
перед запуском обновлённого приложения. Старые сессии получают отдельный token.

Отдельный origin frontend в этой версии не поддерживается: Access-Control-Allow-*
не выставляются, cross-origin preflight отклоняется. CORS не заменяет CSRF.
SameSite используется дополнительно к проверке token и Origin.

## Тела, ошибки и лимиты

JSON: Content-Type application/json; ровно один объект, неизвестные поля
и второй объект/значение запрещены. Ошибка формата — 400, неподдерживаемый
Content-Type — 415, превышение лимита — 413.

| Ресурс | Лимит |
| --- | --- |
| login/register JSON | 16 KiB |
| Обычный JSON | 1 MiB |
| Подтверждение imports JSON | 10 MiB |
| DOCX-файл | 10 MiB |
| Multipart-запрос | 11 MiB |
| Извлекаемый DOCX XML | 20 MiB |
| Одновременные preview и imports | 4 на процесс, общий лимит |
| login | 10 запросов/минуту/IP |
| register | 5 запросов/час/IP |
| preview + imports | 10 запросов/минуту/пользователя, общий счётчик |

Rate limiter хранится в памяти одного backend: не более 10 000 ключей
на limiter, по заполнении новые ключи получают 429 до освобождения истёкших.
Перезапуск обнуляет счётчики. Ответ 429 содержит Retry-After.
IP берётся из RemoteAddr; X-Forwarded-For не доверяется. За reverse proxy
auth-лимит может быть общим для клиентов proxy — перед открытым тестом настройте
дополнительные per-IP лимиты на proxy. Для нескольких backend нужен общий limiter.

Ошибки имеют форму `{"error":"текст"}`. Отсутствие auth — 401, запрет CSRF — 403,
чужой/отсутствующий ресурс — 404. SQL, пароль, token и пути сервера не возвращаются.
Создание ресурсов — 201; 204 всегда без JSON. Login использует одинаковый ответ
401 для неизвестного email и неправильного пароля; сбой PostgreSQL остаётся 500.

## Работа процесса

Логи содержат method, шаблон route (без пользовательского URL/query), status,
duration, X-Request-ID. Безопасный входящий request ID сохраняется,
иначе генерируется новый. Cookie, token, пароли и содержимое документов не логируются.
Panic одного handler перехватывается, клиент получает безопасный 500;
если ответ уже начат, соединение прерывается. Процесс продолжает работать.

HTTP timeouts: заголовки 5 секунд, чтение/запись 30 секунд, idle 60 секунд;
SQL использует context запроса, общий дедлайн запроса — 30 секунд.
Graceful shutdown: до 10 секунд на активные запросы, затем закрытие соединений;
pool закрывается после остановки HTTP-сервера. `/health` — лёгкая проверка,
`/ready` проверяет PostgreSQL с timeout 2 секунды.

Секреты задаются окружением, `.env` и ключи исключены из git и Docker context.
`.env.example` содержит только placeholders. Перед публикацией проверяйте
полную историю git и настройки используемой среды. Если секрет уже опубликован,
его нужно заменить у провайдера/в БД; удаление файла не отзывает секрет.

Основания: [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html),
[OWASP CSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html),
[Go net/http](https://pkg.go.dev/net/http).
