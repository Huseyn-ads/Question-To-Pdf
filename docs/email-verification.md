# Подтверждение email

Эта версия добавляет проверку email к существующей регистрации. Пользователь
регистрируется, получает письмо, открывает ссылку, нажимает «Подтвердить email»
и затем выполняет Login. Google Sign-In сюда не входит.

## Установка

Патч email-verification.patch рассчитан на проект после патчей этапов 11 и 13.
Сначала сохранить свои изменения в Git, затем из корня проекта:

~~~bash
git apply --check /путь/к/email-verification.patch
git apply /путь/к/email-verification.patch
~~~

Если проверка не прошла, не использовать --reject и не удалять файлы: сначала
сверить версию проекта и сообщение Git.

До запуска нового backend применить миграцию:

~~~bash
make migrate-up
~~~

Новая миграция: migrations/20261104180000_add_email_verification.sql.
Она добавляет users.email_verified_at и таблицу email_verification_tokens.
Старые аккаунты сохраняют доступ: их email_verified_at заполняется временем
миграции. Это политика перехода; она не означает фактическую проверку старой почты.
Новые аккаунты всегда начинают с NULL и требуют подтверждения.

Патч меняет .env.example, но не личный .env. Добавить SMTP-настройки в свой
.env вручную. В проекте Makefile читает .env, а go run сам его не загружает.

## Вариант A: Gmail

Для отправителя использовать отдельный почтовый аккаунт проекта. Получатели
могут использовать Gmail, Outlook и другие почтовые сервисы.

1. Включить двухэтапную аутентификацию у Google-аккаунта отправителя.
2. Создать App Password: https://myaccount.google.com/apppasswords.
3. Добавить в локальный .env следующие значения, заменив placeholders:

~~~dotenv
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-sender@gmail.com
SMTP_PASSWORD=replace-with-google-app-password
SMTP_FROM=your-sender@gmail.com
SMTP_TLS=starttls
~~~

SMTP_PASSWORD — пароль приложения, а не обычный пароль Google. Если Google
показывает пароль группами, убрать пробелы. Его нельзя коммитить, публиковать,
присылать в чат или сохранять в Insomnia collection.

Для implicit TLS можно использовать SMTP_PORT=465 и SMTP_TLS=tls.
STARTTLS обязателен в режиме starttls: backend не переходит молча на plaintext.
TLS-сертификат SMTP-сервера проверяется. Docker image содержит CA certificates.

Некоторые Google-аккаунты не позволяют создавать App Password: например,
настройки организации или Advanced Protection могут ограничивать эту возможность.
В таком случае можно использовать другой SMTP-провайдер с теми же переменными.
Инструкция Google: https://support.google.com/accounts/answer/185833.

После настройки перезапустить backend, например:

~~~bash
docker compose up -d --build app
~~~

PUBLIC_ORIGIN должен совпадать с адресом страницы сайта: на нём строится ссылка
из письма. Для локального проекта это, например, http://localhost:8080.
Для публикации нужны реальный HTTPS origin и APP_ENV=production.
В production SMTP_TLS=none запрещён.

## Вариант B: локальная почта для разработки

Mailpit принимает тестовые письма локально. Они не отправляются в настоящий Gmail.

~~~bash
docker compose --profile dev-mail up -d mailpit db
~~~

Добавить в .env:

~~~dotenv
SMTP_HOST=mailpit
SMTP_PORT=1025
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=no-reply@example.test
SMTP_TLS=none
~~~

Эти значения предназначены для backend внутри Docker. Если backend запущен через
go run на компьютере, заменить SMTP_HOST на localhost и передать переменные
окружения процессу. Пример загрузки собственного доверенного .env в терминале:

~~~bash
set -a
source .env
set +a
go run ./cmd/api
~~~

Перед этим в .env также поставить DB_HOST=localhost и DB_PORT равным
DB_HOST_PORT, если PostgreSQL работает в Docker, а backend — на компьютере.

Открыть http://localhost:8025, зарегистрировать новый аккаунт и посмотреть письмо.
Порты Mailpit опубликованы только на 127.0.0.1. Это инструмент локальной разработки;
в публичной среде использовать настоящий SMTP и не включать dev-mail profile.
Документация: https://mailpit.axllent.org/docs/install/docker/.

## HTTP-поведение

Полные payload и ошибки описаны в api.md.

- POST /v1/auth/register: 201, verification_required=true.
- verification_email_sent=true означает принятие письма SMTP relay, а не гарантию
  появления во входящих. При сбое SMTP аккаунт сохраняется, поле равно false.
- POST /v1/auth/login: 403 email verification required только после верного пароля.
  Неверный пароль по-прежнему даёт 401.
- GET /v1/auth/verify-email: встроенная HTML-страница без требования login.
- POST /v1/auth/verify-email: принимает token в JSON и подтверждает email.
- POST /v1/auth/resend-verification: принимает email и возвращает общий ответ 202.
- У User JSON добавлено поле email_verified_at; после подтверждения это дата.
- Подтверждение не создаёт сессию автоматически. Далее нужен Login.

Verify/resend не требуют session CSRF token, поскольку выполняются до login.
Они требуют доверенный Origin/Referer, как Register и Login.
В браузере Origin формируется самим браузером. В Insomnia надо передать его явно.

Письмо содержит ссылку вида:

~~~text
https://your-domain/v1/auth/verify-email#token=<one-time-token>
~~~

Token находится во fragment. Он не отправляется в URL HTTP-запроса и не попадает
в server access logs как query. Страница убирает fragment из адресной строки.
Для подтверждения нужна кнопка; автоматический GET от почтового сканера не
изменяет аккаунт. Страница не загружает сторонние ресурсы и использует CSP.

## Срок, повторное использование и ограничения

Token содержит 32 криптографически случайных байта и кодируется в base64url.
В БД хранится SHA-256 hash. Срок действия — 24 часа; на границе expires_at ссылка
уже недействительна. VerificationLifetime в internal/user/verification.go —
единая константа для срока выдаваемого token.

Подтверждение и расходование token выполняются в короткой транзакции с блокировкой
строки пользователя. Время проверяется после получения блокировки. Два
одновременных подтверждения не могут успешно использовать одну ссылку дважды.

Фактический resend помечает старую ссылку использованной и создаёт новую. При
сбое SMTP новая ссылка может не прийти, а старая уже не действует: нужно снова
запросить письмо после cooldown. Сетевые повторы не запускаются автоматически.

Cooldown выдачи — 60 секунд. Максимум 5 выдач за час на аккаунт, включая Register.
Ограничения хранятся в PostgreSQL и не сбрасываются при рестарте backend.
Ответ 202 одинаков для неизвестного/подтверждённого email и тихого cooldown.
Синхронная SMTP-отправка может давать различия во времени ответа; постоянное
время ответа эта версия не гарантирует.

Дополнительно: Register 5/час/IP, Resend 10/час/IP, Verify 60/час/IP.
IP-лимиты используют существующий limiter одного процесса и сбрасываются после
его рестарта. Ошибка превышения IP-лимита — 429 с Retry-After.

В БД нет открытого token, но он есть в самом письме. Не добавлять письмо, token,
SMTP_PASSWORD или cookie в логи. Новые frontend-страницы также не должны писать
fragment/token в аналитику.

## Ручная проверка

1. Register нового email: 201, email_verified_at=null, cookie отсутствует.
2. Login с правильным паролем до подтверждения: 403.
3. Неверный пароль: 401 независимо от подтверждения.
4. Открыть письмо и страницу: аккаунт ещё не подтверждён.
5. Нажать кнопку: email подтверждён, Login возвращает 200 и cookie.
6. Повторно отправить этот token: 400, подтверждение аккаунта сохраняется.
7. Для другого нового аккаунта запросить resend через 60 секунд: старая ссылка
   даёт 400, новая подтверждает аккаунт.
8. Проверить неизвестный email в resend: тот же общий 202.
9. Остановить локальный Mailpit перед Register: 201 с
   verification_email_sent=false. Запустить Mailpit и запросить resend после cooldown.
10. Проверить просроченный token и одновременное подтверждение тестами.

## Если письмо не пришло

verification_email_sent=false означает сбой отправки, а не попадание письма в спам.
Проверьте, что запущенный backend действительно получил SMTP_HOST,
SMTP_PORT и SMTP_TLS из нужного .env. При go run сам файл .env не читается;
используйте команды загрузки окружения выше. После изменения .env контейнер
нужно пересоздать, а процесс go run — перезапустить.

После патча диагностики лог содержит только безопасный этап:

- SMTP connection failed: DNS, сеть, порт или недоступный SMTP_HOST.
- SMTP STARTTLS failed / SMTP TLS handshake failed: TLS или сертификаты.
- SMTP authentication failed: SMTP_USER или пароль приложения не принят сервером.
- SMTP sender rejected / SMTP recipient rejected: SMTP-сервер отклонил адрес.
- SMTP message acceptance failed: сервер не подтвердил приём тела письма.

Пароль, token, письмо и исходный ответ SMTP-сервера в лог не записываются.
Для Docker смотреть docker compose logs --tail=50 app; для go run — терминал
с запущенным backend. После исправления использовать Resend Verification:
аккаунт уже создан. Между выдачами ссылок действует cooldown 60 секунд.

## Автоматические проверки

~~~bash
go test ./...
go test -race ./...
go vet ./...
~~~

Unit/HTTP-тесты используют фиктивные хранилища, SMTP-тесты — локальный тестовый
SMTP-сервер. Они не отправляют настоящие письма.

Для проверки транзакций на реальном PostgreSQL нужна отдельная тестовая БД:

~~~bash
TEST_DATABASE_URL='postgres://test_user:test_password@localhost:5432/test_db?sslmode=disable' go test ./internal/user -run 'TestVerificationRepository|TestVerificationExpiryCheckedAfterLock' -v
~~~

Это вымышленные credentials: заменить своими тестовыми. Интеграционные тесты
создают отдельную схему, применяют две нужные миграции внутри неё, проверяют
откат регистрации, cooldown/resend, одноразовое подтверждение и expiry. Затем
удаляют только свою схему. Без TEST_DATABASE_URL эти тесты явно пропускаются.
