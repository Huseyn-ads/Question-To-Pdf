# API-контракт Question-To-Quiz v1

Этот файл — основной контракт между backend и frontend для первой версии.
Он описывает фактические HTTP-маршруты, JSON-поля, авторизацию, CSRF,
таймер попытки и поведение повторных запросов.

## Быстрый старт frontend

1. Использовать относительные URL `/v1/...` на origin страницы. Для `fetch`
   использовать `credentials: "same-origin"`.
2. При загрузке приложения выполнить `GET /v1/auth/me`. Ответ `200` даёт профиль,
   ответ `401` означает, что нужно показать вход. После обычного или Google-входа
   браузер хранит HttpOnly cookie `session`.
3. При регистрации по email обработать `201` как создание аккаунта и показать
   экран ожидания подтверждения. При `verification_email_sent: false` предложить
   повторную отправку через `/v1/auth/resend-verification`. После подтверждения
   пользователь входит по паролю. Google-вход запускается ссылкой на
   `/v1/auth/google/start`.
4. Для действующей сессии получить `csrf_token` из `GET /v1/auth/csrf` и держать
   его в памяти. Передавать `X-CSRF-Token` в защищённых запросах `POST`, `PUT`,
   `PATCH` и `DELETE`. При logout очистить профиль и CSRF token.
5. При `401` защищённого маршрута вернуться ко входу. Ответ `401` самого Login
   обработать как ошибку учётных данных, а `403` с `email verification required`
   как приглашение подтвердить email.
6. Для восстановления экрана попытки использовать `GET /v1/attempts/{attemptID}`.
   Для повторного открытия результата использовать `GET /v1/attempts/{attemptID}/result`.

### Локальный запуск с frontend dev server

Если frontend открыт на `http://localhost:5173`, браузер обращается к API на
этом же origin. Настрой прокси всех путей `/v1/...` на backend с сохранением
пути, включая HTML-подтверждение и callback Google.

| Настройка | Пример |
| --- | --- |
| Адрес frontend в браузере | `http://localhost:5173` |
| Назначение прокси `/v1/...` | `http://localhost:8080` |
| `PUBLIC_ORIGIN` в backend `.env` | `http://localhost:5173` |
| Authorized redirect URI в Google для этого запуска | `http://localhost:5173/v1/auth/google/callback` |
| `GOOGLE_REDIRECT_URL`, если задан в backend `.env` | `http://localhost:5173/v1/auth/google/callback` |

Для проверки backend напрямую используй `PUBLIC_ORIGIN=http://localhost:8080`
и callback с тем же портом. Для каждого режима в Google нужен точный callback URI.
Ссылки в письмах также строятся от текущего `PUBLIC_ORIGIN`.

Готовая коллекция: [Question-To-Quiz.insomnia.json](insomnia/Question-To-Quiz.insomnia.json).

## Общие соглашения

### URL, JSON и идентификаторы

- Основной API prefix: `/v1`.
- Локальный пример base URL: `http://localhost:8080`.
- Большинство JSON-полей используют `snake_case`. Точный регистр смотри в примере
  конкретного маршрута. В полном квизе поле называется `Questions`.
- Идентификаторы ресурсов — UUID, передаваемые JSON-строками.
- Для запроса с JSON-телом указывать `Content-Type: application/json`.
- Сервер принимает ровно один JSON-объект. Неизвестные поля, массив вместо
  объекта и второй JSON-объект дают ошибку.
- Создание возвращает `201 Created`.
- Успешное удаление и logout возвращают `204 No Content` без тела. Обработчик
  ответа завершает такой запрос без вызова `response.json()`.
- Ответы JSON имеют `Content-Type: application/json; charset=utf-8`.

Примеры используют вымышленные UUID и данные:

| Переменная | Пример |
| --- | --- |
| `bankID` | `22222222-2222-4222-8222-222222222222` |
| `questionID` | `33333333-3333-4333-8333-333333333333` |
| `quizID` | `66666666-6666-4666-8666-666666666666` |
| `quizQuestionID` | `77777777-7777-4777-8777-777777777777` |
| `optionID` | `88888888-8888-4888-8888-888888888888` |
| `attemptID` | `aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa` |

### Время и серверный таймер

Время передаётся по ISO 8601/RFC 3339 с UTC или явным offset, например
`2026-11-02T12:00:00Z`.

`deadline_at: null` означает, что у попытки нет таймера. При создании quiz
`time_limit_minutes: 0` создаёт такие попытки. Положительное значение от 1
до 180 задаёт лимит в минутах.

Источник истины — серверное время. У подробной попытки есть `server_time`.
Frontend вычисляет оставшееся время как `deadline_at - server_time`, а затем
уменьшает его локально для отображения. Reload и перезапуск backend не меняют
`started_at` или `deadline_at`.

Граница истечения едина: при `server_time >= deadline_at` попытка просрочена.
После этого `SaveAnswer` возвращает `409`, даже если в БД статус ещё не был
обновлён отдельным фоновым процессом. Фоновый timer или cron не требуется.

`finished_at` — сохранённое сервером время закрытия попытки. Frontend не
вычисляет его из `started_at` и `deadline_at`.

### Сессия, Origin и CSRF

Backend использует cookie `session` с `Path=/`, `HttpOnly`, `SameSite=Lax`
и `Secure=true` в production HTTPS. Session token не возвращается в JSON.

`POST /v1/auth/register`, `POST /v1/auth/login`,
`POST /v1/auth/verify-email` и `POST /v1/auth/resend-verification` не требуют login или
`X-CSRF-Token`, но, как и все изменяющие запросы, требуют точный
`Origin: PUBLIC_ORIGIN` либо доверенный `Referer`.

После login frontend вызывает `GET /v1/auth/csrf`:

~~~json
{
  "csrf_token": "<csrf-token>"
}
~~~

Все защищённые `POST`, `PUT`, `PATCH` и `DELETE` требуют:

~~~http
Origin: http://localhost:8080
X-CSRF-Token: <csrf-token>
~~~

CSRF token связан с конкретной сессией и не является session token. Его следует
держать в памяти, получать заново при восстановлении сессии после reload или
после нового входа и удалять после logout. Заголовки `Origin` и `Referer` для
браузерных запросов формирует браузер. Frontend настраивает свой origin и прокси.
Safe-методы `GET` и `HEAD` не требуют заголовок `X-CSRF-Token`.

Frontend и API в MVP работают на одном origin. Backend не выдаёт permissive
CORS headers и отклоняет посторонний origin.

### Ошибки

Все прикладные ошибки имеют одну форму:

~~~json
{
  "error": "понятное сообщение"
}
~~~

Общие статусы:

| Status | Значение для frontend |
| --- | --- |
| `400` | Неверный UUID, query parameter, JSON или проверка полей |
| `401` | Нет действующей сессии на защищённом маршруте или неверные учётные данные при Login |
| `403` | Неверный CSRF/origin, неподтверждённый email при Login или неподходящий профиль Google |
| `404` | Ресурс не существует или принадлежит другому пользователю |
| `405` | Метод не поддерживается; сервер также возвращает `Allow` |
| `409` | Конфликт состояния, например попытка закрыта |
| `413` | Тело или файл превышает лимит |
| `415` | Неверный `Content-Type` или не DOCX |
| `422` | DOCX повреждён или содержит неподдерживаемые formula/media |
| `429` | Rate limit/capacity; учитывать `Retry-After`, если он есть |
| `500` | Безопасное `server error`, без SQL, пароля и пути сервера |
| `502` | Callback Google не смог обменять или проверить токен |
| `503` | PostgreSQL временно недоступен на `/ready` или Google-вход отключён |

### Владение ресурсами

Пользователь видит и изменяет только свои банки, questions, quizzes и attempts.
Отсутствующий и чужой ресурс намеренно дают одинаковый `404`. Frontend не должен
делать вывод о существовании чужого UUID.

### Списки и пагинация

`GET /v1/quizzes` возвращает объект пагинации.

~~~json
{
  "items": [],
  "limit": 20,
  "offset": 0
}
~~~

`GET /v1/quizzes/{quizID}/attempts` возвращает массив попыток напрямую и также
принимает `limit` и `offset` в query. Для обоих списков `limit` по умолчанию 20,
допустим от 1 до 100, а `offset` по умолчанию 0 и неотрицателен. Последняя страница
определяется по длине массива меньше запрошенного `limit`.

Сортировка `GET /v1/quizzes` по `created_at DESC, id DESC`, а истории попыток по
`started_at DESC, id DESC`. Списки банков и вопросов возвращают непагинированный
JSON-массив напрямую.

### Полный quiz, метаданные и правильность

- `POST /v1/question-banks/{bankID}/quizzes` возвращает только метаданные
  созданного quiz.
- `GET /v1/quizzes` возвращает только метаданные.
- `GET /v1/quizzes/{quizID}` возвращает полный snapshot вопросов и вариантов,
  но не возвращает `is_correct`.
- `GET /v1/attempts/{attemptID}` возвращает полный snapshot и выбранные ID,
  но не раскрывает правильность.
- Авторские `is_correct` и итог по каждому вопросу доступны только из
  `POST /v1/attempts/{attemptID}/finish` и
  `GET /v1/attempts/{attemptID}/result` после закрытия.

Quiz хранит копии вопросов и вариантов. Изменение исходного банка после создания
quiz не меняет его состав или результат попытки.

## Карта маршрутов

| Method и path | Login | CSRF | Назначение |
| --- | --- | --- | --- |
| `POST /v1/auth/register` | нет | нет, нужен Origin | Регистрация |
| `GET /v1/auth/verify-email` | нет | нет | Страница подтверждения (HTML) |
| `POST /v1/auth/verify-email` | нет | нет, нужен Origin | Подтвердить email по token |
| `POST /v1/auth/resend-verification` | нет | нет, нужен Origin | Повторное письмо |
| `POST /v1/auth/login` | нет | нет, нужен Origin | Login и session cookie |
| `GET /v1/auth/google/start` | нет | OAuth state и PKCE | Начать вход в Google |
| `GET /v1/auth/google/callback` | нет | OAuth state, PKCE и nonce | Выдать session cookie |
| `GET /v1/auth/me` | да | нет | Текущий пользователь |
| `GET /v1/auth/csrf` | да | нет | CSRF token сессии |
| `POST /v1/auth/logout` | да | да | Удалить сессию |
| `GET /v1/question-banks` | да | нет | Список банков |
| `POST /v1/question-banks` | да | да | Создать банк |
| `GET /v1/question-banks/{bankID}` | да | нет | Получить банк |
| `PATCH /v1/question-banks/{bankID}` | да | да | Изменить банк |
| `DELETE /v1/question-banks/{bankID}` | да | да | Удалить банк |
| `GET /v1/question-banks/{bankID}/questions` | да | нет | Вопросы банка |
| `POST /v1/question-banks/{bankID}/questions` | да | да | Создать вопрос |
| `GET /v1/question-banks/{bankID}/questions/{questionID}` | да | нет | Получить вопрос |
| `PUT /v1/question-banks/{bankID}/questions/{questionID}` | да | да | Полностью заменить вопрос |
| `DELETE /v1/question-banks/{bankID}/questions/{questionID}` | да | да | Удалить вопрос |
| `POST /v1/question-banks/{bankID}/imports/preview` | да | да | Preview DOCX |
| `POST /v1/question-banks/{bankID}/imports` | да | да | Подтвердить import |
| `POST /v1/question-banks/{bankID}/quizzes` | да | да | Создать quiz snapshot |
| `GET /v1/quizzes` | да | нет | Список quizzes |
| `GET /v1/quizzes/{quizID}` | да | нет | Полный quiz snapshot |
| `DELETE /v1/quizzes/{quizID}` | да | да | Удалить quiz |
| `POST /v1/quizzes/{quizID}/attempts` | да | да | Начать/восстановить попытку |
| `GET /v1/quizzes/{quizID}/attempts` | да | нет | История попыток |
| `GET /v1/attempts/{attemptID}` | да | нет | Восстановить экран попытки |
| `PUT /v1/attempts/{attemptID}/answers/{quizQuestionID}` | да | да | Сохранить выбор |
| `POST /v1/attempts/{attemptID}/finish` | да | да | Закрыть попытку |
| `GET /v1/attempts/{attemptID}/result` | да | нет | Получить результат |
| `GET /health` | нет | нет | Liveness |
| `GET /ready` | нет | нет | Readiness PostgreSQL |

## Auth

### POST /v1/auth/register

Login: нет. CSRF: нет, но доверенный `Origin` обязателен. Path/query: нет.

Body, JSON до 16 KiB:

~~~json
{
  "email": "student@example.com",
  "name": "Demo Student",
  "password": "<enter-a-local-test-password>"
}
~~~

Email нормализуется в lowercase. Имя обязательно. Пароль: минимум 8 Unicode
символов и максимум 72 байта UTF-8.

Успех `201`:

~~~json
{
  "id": "11111111-1111-4111-8111-111111111111",
  "email": "student@example.com",
  "name": "Demo Student",
  "created_at": "2026-11-04T10:00:00Z",
  "email_verified_at": null,
  "verification_required": true,
  "verification_email_sent": true
}
~~~

Ошибки: `400` validation/JSON, `409` email уже занят, `413`, `415`, `429`, `500`.
Регистрация не создаёт session cookie. Повтор с тем же email возвращает `409`.
В одной транзакции создаются аккаунт и hash одноразового token. Письмо отправляется
после commit. `verification_email_sent: true` означает, что SMTP relay принял письмо,
а не гарантирует его появление во входящих.
При сбое SMTP аккаунт остаётся созданным, ответ `201` содержит
`verification_email_sent: false`. Нужно показать кнопку повторной отправки;
повторять Register для этого не нужно.

### GET /v1/auth/verify-email

Login/CSRF: нет. Body/path/query: нет. Это HTML, не JSON API.

Письмо содержит ссылку:

~~~text
http://localhost:8080/v1/auth/verify-email#token=<one-time-token>
~~~

Фрагмент `#token=...` не отправляется HTTP-серверу в URL. Страница читает его,
убирает из адресной строки и показывает кнопку «Подтвердить email».
Только нажатие кнопки отправляет POST ниже. Простое открытие ссылки, включая
предпросмотр письма сканером, не подтверждает аккаунт и не расходует token.
Успех `200 text/html`; повторное открытие страницы безопасно.
Эту HTML-страницу отдаёт backend. Для браузера с frontend dev server путь
`/v1/auth/verify-email` должен попадать в тот же прокси, что и остальные `/v1/...`.

### POST /v1/auth/verify-email

Login: нет. Session CSRF: нет, но доверенный `Origin` обязателен.
Path/query: нет. JSON до 16 KiB:

~~~json
{"token":"<one-time-token-from-email>"}
~~~

Успех `200`:

~~~json
{"message":"email verified; you can now log in"}
~~~

Token случайный, 32 байта, base64url, хранится в БД только как SHA-256 hash.
Срок — 24 часа от выдачи. Подтверждение и расходование token атомарны.
Неверная, истёкшая, заменённая и уже использованная ссылка дают одинаковый
`400` с `verification link is invalid or expired`.
Вторая попытка подтверждения возвращает `400`, но аккаунт остаётся подтверждённым.
Остальные ошибки: `403` Origin, `413`, `415`, `429`, `500`.
Подтверждение не создаёт сессию; затем пользователь выполняет Login.

### POST /v1/auth/resend-verification

Login: нет. Session CSRF: нет, но доверенный `Origin` обязателен.
Path/query: нет. JSON до 16 KiB:

~~~json
{"email":"student@example.com"}
~~~

Ответ `202`:

~~~json
{"message":"if verification is needed, an email will be sent; please wait before trying again"}
~~~

Одинаковый статус и текст возвращаются для отсутствующего, уже подтверждённого,
ожидающего подтверждения аккаунта и при достижении лимита на email. Этот ответ
также не гарантирует доставку письма; синхронная отправка может влиять на время ответа.
При фактической новой выдаче старая ссылка становится недействительной.
Cooldown — 60 секунд, максимум 5 выдач в час на аккаунт, включая Register.
Эти ограничения хранятся в PostgreSQL и переживают restart. Ошибки:
`400` invalid email/JSON, `403` Origin, `413`, `415`,
`429` общий лимит на IP, `500` ошибка хранилища.

### POST /v1/auth/login

Login: нет. CSRF: нет, но доверенный `Origin` обязателен. Path/query: нет.

~~~json
{
  "email": "student@example.com",
  "password": "<enter-a-local-test-password>"
}
~~~

Успех `200` устанавливает HttpOnly cookie `session` и возвращает User:

~~~json
{
  "id": "11111111-1111-4111-8111-111111111111",
  "email": "student@example.com",
  "name": "Demo Student",
  "created_at": "2026-11-04T10:00:00Z",
  "email_verified_at": "2026-11-04T10:05:00Z"
}
~~~

Session token в теле отсутствует.

Ошибки: `400` JSON, `401` одинаково для неизвестного email и неверного пароля,
`403` с `email verification required` для правильного пароля при неподтверждённом
email, `413`, `415`, `429`, `500`. Проверка подтверждения идёт после проверки пароля.
Каждый успешный login создаёт новую сессию и новый
CSRF token; повторный login не возвращает старый token.

### Вход через Google

В backend `.env` задай `GOOGLE_CLIENT_ID` и `GOOGLE_CLIENT_SECRET` OAuth-клиента
типа Web application. Backend строит callback из `PUBLIC_ORIGIN` и
`/v1/auth/google/callback`. В Google Auth Platform добавь этот адрес в **Authorized
redirect URIs**. Если задан `GOOGLE_REDIRECT_URL`, он должен точно совпадать с этим
адресом. Для запуска нужны применённые миграции, включая `add_google_login`.

#### GET /v1/auth/google/start

Кнопка входа запускает навигацию браузера на `/v1/auth/google/start`, например
через ссылку `<a href="/v1/auth/google/start">Войти через Google</a>`.
Запрос выполняется без тела и параметров. Ответ `302` устанавливает временную
HttpOnly cookie на пять минут и перенаправляет браузер в Google. Каждый новый
Start заменяет предыдущую попытку входа в этом браузере. При пустых Google
credentials маршрут отвечает `503`. Превышение лимита запросов даёт `429`.

#### GET /v1/auth/google/callback

Браузер возвращается с параметрами `code` и `state` или с отказом Google. Тело запроса отсутствует. Backend проверяет `state` из временной cookie, обменивает код с PKCE и проверяет подпись ID token, `aud`, `iss`, срок действия и `nonce`.

Первый успешный вход с подтверждённым `@gmail.com` создаёт пользователя и заполняет `email_verified_at`. Повторный вход находит его по Google `sub`, сохраняет локальный `id` и выдаёт новую сессию. Если email уже принадлежит аккаунту с другим способом входа, callback отвечает `409`. Для такого аккаунта используй существующий способ входа.

Успех устанавливает обычную HttpOnly cookie `session` и отвечает `303` с
`Location: /auth/complete`. Это страница frontend того же origin: она загружает
`/v1/auth/me` и `/v1/auth/csrf` так же, как после входа по паролю, и возвращает
пользователя в интерфейс. Параметры `code` и `state` обрабатывает callback backend.
При ошибке callback по-прежнему отвечает JSON `{ "error": "..." }`.

- `400` при отмене, просроченной попытке или несовпадении `state` или `nonce`.
  Начни новый вход через `/v1/auth/google/start`.
- `403` для нового аккаунта без подтверждённого Gmail.
- `409` для уже занятого email.
- `429` при превышении лимита, `503` при выключенном Google login, `502` при сбое Google или ошибке проверки ID token, `500` при ошибке backend.

### GET /v1/auth/me

Login: да. CSRF: нет. Body/path/query: нет.

~~~http
GET /v1/auth/me
~~~

Успех `200` — User JSON из Login. Ошибки: `401`, `500`. Запрос безопасно
повторять.

### GET /v1/auth/csrf

Login: да. CSRF: нет. Body/path/query: нет.

Успех `200`:

~~~json
{
  "csrf_token": "<csrf-token>"
}
~~~

Ошибки: `401`, `500`. Ответ имеет `Cache-Control: no-store`. Запрос можно
повторять; в рамках одной сессии возвращается тот же token.

### POST /v1/auth/logout

Login: да. CSRF: да. Body/path/query: нет.

~~~http
POST /v1/auth/logout
Origin: http://localhost:8080
X-CSRF-Token: <csrf-token>
~~~

Успех `204`, тело отсутствует. Сервер удаляет сессию и очищает cookie.
Ошибки: `401`, `403`, `500`. Повтор после успешного logout возвращает `401`.

## Банки вопросов

Объект банка:

~~~json
{
  "id": "22222222-2222-4222-8222-222222222222",
  "title": "История",
  "description": "Вопросы к экзамену",
  "created_at": "2026-11-02T10:05:00Z",
  "updated_at": "2026-11-02T10:05:00Z"
}
~~~

### GET /v1/question-banks

Login: да. CSRF: нет. Body/path/query: нет.

~~~http
GET /v1/question-banks
~~~

Успех `200` — непагинированный массив объектов банка, `[]` если банков нет.
Ошибки: `401`, `500`. Запрос безопасно повторять.

### POST /v1/question-banks

Login: да. CSRF: да. Path/query: нет. JSON до 1 MiB:

~~~json
{
  "title": "История",
  "description": "Вопросы к экзамену"
}
~~~

Успех `201` — объект банка. `title` обязателен, после trim не длиннее 150
Unicode символов. Ошибки: `400`, `401`, `403`, `413`, `415`, `500`.
Idempotency key нет: повтор создаёт ещё один банк.

### GET /v1/question-banks/{bankID}

Login: да. CSRF: нет. `bankID` — UUID.

~~~http
GET /v1/question-banks/22222222-2222-4222-8222-222222222222
~~~

Успех `200` — объект банка. Ошибки: `400` UUID, `401`, `404`, `500`.
Запрос безопасно повторять.

### PATCH /v1/question-banks/{bankID}

Login: да. CSRF: да. `bankID` — UUID. JSON до 1 MiB. Нужен хотя бы один ключ:

~~~json
{
  "description": "Обновлённое описание"
}
~~~

`title` и `description` заменяются целиком. `null` не считается новым значением.
Успех `200` — обновлённый объект банка. Ошибки: `400`, `401`, `403`, `404`,
`413`, `415`, `500`. Повтор одного payload приводит к тому же содержимому,
но `updated_at` может измениться.

### DELETE /v1/question-banks/{bankID}

Login: да. CSRF: да. `bankID` — UUID. Body/query: нет.

Успех `204`, тело отсутствует. Ошибки: `400`, `401`, `403`, `404`, `500`.
Удаление каскадно удаляет связанные данные. Повтор возвращает `404`.

## Вопросы банка

Объект авторского вопроса:

~~~json
{
  "id": "33333333-3333-4333-8333-333333333333",
  "question_bank_id": "22222222-2222-4222-8222-222222222222",
  "text": "В каком году началась Вторая мировая война?",
  "options": [
    {
      "id": "44444444-4444-4444-8444-444444444444",
      "text": "1939",
      "position": 1,
      "is_correct": true
    },
    {
      "id": "55555555-5555-4555-8555-555555555555",
      "text": "1941",
      "position": 2,
      "is_correct": false
    }
  ],
  "created_at": "2026-11-02T10:10:00Z",
  "updated_at": "2026-11-02T10:10:00Z"
}
~~~

Эти authoring endpoints показывают `is_correct` владельцу банка. В student
attempt это поле скрыто до закрытия.

### GET /v1/question-banks/{bankID}/questions

Login: да. CSRF: нет. `bankID` — UUID. Query/body: нет.

Успех `200` — непагинированный массив объектов вопроса. Ошибки: `400` UUID,
`401`, `404`, `500`. Запрос безопасно повторять.

### POST /v1/question-banks/{bankID}/questions

Login: да. CSRF: да. `bankID` — UUID. JSON до 1 MiB:

~~~json
{
  "text": "В каком году началась Вторая мировая война?",
  "options": [
    {"text": "1939", "is_correct": true},
    {"text": "1941", "is_correct": false}
  ]
}
~~~

Для ручного создания нужны минимум два непустых уникальных варианта и ровно один
`is_correct: true`. Позиции назначает сервер по порядку массива.

Успех `201` — полный объект вопроса. Ошибки: `400`, `401`, `403`, `404`,
`413`, `415`, `500`. Повтор создаёт ещё один вопрос.

### GET /v1/question-banks/{bankID}/questions/{questionID}

Login: да. CSRF: нет. Оба path parameter — UUID.

Успех `200` — полный объект вопроса. Ошибки: `400`, `401`, `404`, `500`.
Чужой `questionID` или вопрос из другого банка дают `404`. Запрос повторяемый.

### PUT /v1/question-banks/{bankID}/questions/{questionID}

Login: да. CSRF: да. Оба path parameter — UUID. JSON совпадает с create:

~~~json
{
  "text": "В каком году началась Вторая мировая война?",
  "options": [
    {"text": "1939", "is_correct": true},
    {"text": "1941", "is_correct": false}
  ]
}
~~~

Это полная замена текста и всех вариантов, не частичный patch. Успех `200` —
обновлённый вопрос. Ошибки: `400`, `401`, `403`, `404`, `413`, `415`, `500`.
Повтор сохраняет то же содержимое, но варианты могут получить новые UUID.

### DELETE /v1/question-banks/{bankID}/questions/{questionID}

Login: да. CSRF: да. Оба path parameter — UUID. Body/query: нет.

Успех `204`. Ошибки: `400`, `401`, `403`, `404`, `500`.
Повтор возвращает `404`.

## Import DOCX

Поддерживаются тестовые документы Word с вариантами ответа. Формулы
(Office Math), изображения и embedded objects возвращают `422`; универсальный
Word/PDF parser не обещается.

Распознаваемая разметка:

- номера вопросов и метки вариантов, набранные текстом (`12.`, `12)`, `B)`,
  `c.`) или созданные списками Word (номер есть только в `numbering.xml`);
- варианты по одному в абзаце, несколько в одной строке (`a) x  b) y`),
  через разрывы строк в одном абзаце или начиная со строки вопроса;
- текст между строкой вопроса и вариантами (предложение, код, пронумерованные
  утверждения, таблица сопоставления) входит в `text` вопроса через `\n`;
  автоматические номера таких строк сохраняются (`1. a white coat  a. mechanic`);
- варианты, пронумерованные цифрами `1)`–`5)`, если буквенных нет;
- пронумерованные заголовки разделов без вариантов, заголовки тем, номера
  страниц, скрытый текст и пометки сложности/веса (`(a)`, `(o)`, `(ç)`, `A`/`O`/`Ç`
  в конце вопроса, `(çəki2)`, `(asan tipli)`) не импортируются.

Порядок вопросов и вариантов сохраняется как в документе, метки вариантов
(`label`) — как их видит читатель документа, включая ошибки автора (повтор
метки, `F)` после `E)`).

### POST /v1/question-banks/{bankID}/imports/preview

Login: да. CSRF: да. `bankID` — UUID. Body: `multipart/form-data`, поле
`file` содержит один `.docx`.

Необязательный параметр `answer_mode` — поле формы или query parameter:

| Значение | Как выставляется `is_correct` |
| --- | --- |
| `auto` (по умолчанию) | По явному ключу в документе, иначе по авторской отметке |
| `all_a` | В каждом вопросе правильным считается вариант с исходной меткой `A` |

`all_a` предназначен для документов без разметки, где пользователь знает, что
все ответы — A. Если в вопросе нет варианта `A` или он встречается дважды,
вариант не отмечается и в `issues` появляется замечание с номером вопроса.
Если явный ключ документа или отметка автора указывает другой вариант, `A`
всё равно отмечается, а в `issues` появляется `answer_key_conflict`. Передача
`answer_mode` и в форме, и в query с разными значениями, а также другое
значение дают `400`. Для frontend: переключатель «все ответы A» отправляет
`FormData` с полем `answer_mode=all_a`.

Источник правильного ответа в режиме `auto`, по убыванию приоритета:

1. строка ключа у вопроса: `Cavab: C`, `Düzgün cavab - (b)`, `Answer: B`,
   `Ответ: A`;
2. отдельный ключ ответов: заголовок `Cavablar`, `Açar`, `Answer key`,
   `Ответы`, `Ключ` и т. п., после него строки `1-A, 2-C`, `3) B`, нумерованный
   список букв или таблица «номер | буква»; без заголовка ключом считается
   строка минимум из трёх пар с заглавными буквами (`1-A 2-C 3-B`);
3. явная фраза в документе, что правильный ответ везде A (`Bütün suallarda
   düzgün cavab A variantıdır`, `Правильный ответ везде — A`,
   `The correct answer is always A`);
4. авторская отметка варианта: жирный, выделение цветом (highlight), цвет
   шрифта, подчёркивание или заливка. Для документа выбирается одна отметка —
   та, что выделяет ровно один вариант в наибольшем числе вопросов. Оттенки
   одного цвета (`FF0000`, `EE0000`) считаются одной отметкой. Если отметкой
   оформлены все варианты вопроса, она не считается ответом.

Если источника нет, варианты остаются с `is_correct: false` и появляется
`correct_answer_missing`. Если ключ противоречит отметке автора, используется
ключ и появляется `answer_key_conflict`.

Пример формы:

~~~http
POST /v1/question-banks/22222222-2222-4222-8222-222222222222/imports/preview
Origin: http://localhost:8080
X-CSRF-Token: <csrf-token>
Content-Type: multipart/form-data; boundary=...
~~~

Успех `200`:

~~~json
{
  "questions_count": 1,
  "can_import": true,
  "issues": [],
  "questions": [
    {
      "source_number": 1,
      "text": "Столица Франции?",
      "options": [
        {"label": "A", "text": "Париж", "is_correct": true},
        {"label": "B", "text": "Лион", "is_correct": false}
      ]
    }
  ]
}
~~~

`issues` может содержать `source_number`, `field`, `code`, `message`.
В `issues` поля `source_number` и `field` могут отсутствовать; без
`source_number` замечание относится ко всему документу.

`source_number` — номер вопроса из документа. Если нумерация в документе
перезапускается (разделы, юниты) или повторяется, `source_number` — позиция
вопроса в документе (1…N), а в `issues` есть `source_numbers_renumbered`.

Коды замечаний парсера (кроме проверок черновика вроде `not_enough_options`,
`duplicate_option_text`, `duplicate_option_label`):

| `code` | Значение |
| --- | --- |
| `correct_answer_missing` | Нет отметки и ключа; варианты не отмечены |
| `correct_answer_ambiguous` | Отмечено несколько вариантов или метка ключа повторяется |
| `answer_key_invalid` | Ключ указывает букву, которой нет среди вариантов |
| `answer_key_conflict` | Ключ, отметка автора или `all_a` указывают разные варианты |
| `answer_mode_label_missing` | `all_a`: в вопросе нет варианта `A` |
| `answer_mode_label_duplicate` | `all_a`: метка `A` встречается несколько раз |
| `empty_option_skipped` | Пустой пронумерованный вариант пропущен |
| `content_skipped` | Пронумерованный текст без вариантов (заголовок, список открытых вопросов) не импортирован |
| `source_numbers_renumbered` | Нумерация перезапускается; `source_number` — позиция |

Для вопроса с замечанием о правильном ответе отдельное замечание
`exactly_one_correct_option` не дублируется. Покажи замечания
перед подтверждением. Текущий Preview возвращает `can_import: true` и при
непустом `issues`. Это замечания к черновику, который пользователь может
просмотреть и отредактировать. При пустом `questions` предложи выбрать другой
документ. Для браузерного запроса добавь файл в `FormData` под ключом `file`
и дай браузеру сформировать `Content-Type` с границей частей. Preview можно
повторять, он не записывает вопросы в банк.

Ошибки: `400` form/file или `answer_mode`, `401`, `403`, `404`, `413` (request 11 MiB или
file 10 MiB), `415` не DOCX, `422` повреждённый/неподдерживаемый DOCX,
`429`, `500`. Извлекаемый XML ограничен 20 MiB.

### POST /v1/question-banks/{bankID}/imports

Login: да. CSRF: да. `bankID` — UUID. JSON до 10 MiB. Обычно frontend
отправляет отредактированный массив `questions` из preview:

~~~json
{
  "questions": [
    {
      "source_number": 1,
      "text": "Столица Франции?",
      "options": [
        {"label": "A", "text": "Париж", "is_correct": true},
        {"label": "B", "text": "Лион", "is_correct": false}
      ]
    }
  ]
}
~~~

Успех `201`:

~~~json
{
  "imported_questions": 1,
  "imported_options": 2
}
~~~

Ошибки: `400`, `401`, `403`, `404`, `413`, `415`, `429`, `500`.
Все вопросы и варианты одного запроса записываются одной транзакцией.
Список вопросов банка возвращает импортированные вопросы в порядке массива
`questions`, варианты — в порядке `options`.

Повторный `POST /imports` добавляет ещё один набор. Автоматической
дедупликации и idempotency key нет. Frontend не должен повторять запрос после
неясного network failure без подтверждения пользователя.

## Quizzes

### POST /v1/question-banks/{bankID}/quizzes

Login: да. CSRF: да. `bankID` — UUID. JSON до 1 MiB:

~~~json
{
  "question_count": 1,
  "time_limit_minutes": 30,
  "shuffle_options": true
}
~~~

`question_count` должен быть положительным и не больше числа вопросов банка.
`time_limit_minutes` допустим от 0 до 180; `0` означает отсутствие timer.

Успех `201`, только метаданные:

~~~json
{
  "id": "66666666-6666-4666-8666-666666666666",
  "question_bank_id": "22222222-2222-4222-8222-222222222222",
  "question_count": 1,
  "created_at": "2026-11-02T10:20:00Z"
}
~~~

Сервер случайно выбирает вопросы и сохраняет snapshot. Ошибки: `400`, `401`,
`403`, `404`, `413`, `415`, `500`. Повтор создаёт новый quiz и новую случайную
подборку.

### GET /v1/quizzes

Login: да. CSRF: нет. Query: `limit` и `offset`.

~~~http
GET /v1/quizzes?limit=20&offset=0
~~~

Успех `200`:

~~~json
{
  "items": [
    {
      "id": "66666666-6666-4666-8666-666666666666",
      "question_bank_id": "22222222-2222-4222-8222-222222222222",
      "question_count": 1,
      "time_limit_minutes": 30,
      "shuffle_options": true,
      "created_at": "2026-11-02T10:20:00Z"
    }
  ],
  "limit": 20,
  "offset": 0
}
~~~

Ошибки: `400` query, `401`, `500`. Запрос безопасно повторять.

### GET /v1/quizzes/{quizID}

Login: да. CSRF: нет. `quizID` — UUID.

Успех `200`, полный snapshot без правильности:

~~~json
{
  "id": "66666666-6666-4666-8666-666666666666",
  "question_bank_id": "22222222-2222-4222-8222-222222222222",
  "question_count": 1,
  "time_limit_minutes": 30,
  "shuffle_options": true,
  "created_at": "2026-11-02T10:20:00Z",
  "Questions": [
    {
      "id": "77777777-7777-4777-8777-777777777777",
      "question_id": "66666666-6666-4666-8666-666666666666",
      "text": "Столица Франции?",
      "position": 1,
      "quiz_answer_options": [
        {
          "id": "88888888-8888-4888-8888-888888888888",
          "quiz_question_id": "77777777-7777-4777-8777-777777777777",
          "text": "Париж",
          "position": 1
        }
      ]
    }
  ]
}
~~~

В этом ответе `Questions[].question_id` содержит UUID квиза. UUID вопроса
снимка находится в `Questions[].id`. В подробной попытке коллекции называются
`questions` и `options`, как показано в разделе Attempts.

Ошибки: `400`, `401`, `404`, `500`. Запрос безопасно повторять.

### DELETE /v1/quizzes/{quizID}

Login: да. CSRF: да. `quizID` — UUID. Body/query: нет.

Успех `204`. Ошибки: `400`, `401`, `403`, `404`, `500`.
Удаляются snapshot и связанные attempts. Повтор возвращает `404`.

## Attempts, ответы и результат

Статусы: `in_progress`, `completed`, `expired`.

Подробная активная попытка:

~~~json
{
  "id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "quiz_id": "66666666-6666-4666-8666-666666666666",
  "status": "in_progress",
  "started_at": "2026-11-02T10:30:00Z",
  "deadline_at": "2026-11-02T11:00:00Z",
  "finished_at": null,
  "server_time": "2026-11-02T10:30:05Z",
  "questions": [
    {
      "id": "77777777-7777-4777-8777-777777777777",
      "text": "Столица Франции?",
      "position": 1,
      "options": [
        {
          "id": "88888888-8888-4888-8888-888888888888",
          "text": "Париж",
          "position": 1
        }
      ]
    }
  ],
  "answers": []
}
~~~

### POST /v1/quizzes/{quizID}/attempts

Login: да. CSRF: да. `quizID` — UUID. Body/query: нет.

Успех:

- `201` — создана новая попытка;
- `200` — уже есть активная попытка этого пользователя для этого quiz.

В обоих случаях тело — подробная попытка выше. Повтор во время `in_progress`
возвращает тот же `attemptID`, `started_at` и `deadline_at`. После `completed`
или `expired` следующий Start создаёт новую попытку с новым временем и без
переноса ответов. Для новой случайной подборки нужно создать новый quiz.

Ошибки: `400`, `401`, `403`, `404`, `500`.

### GET /v1/attempts/{attemptID}

Login: да. CSRF: нет. `attemptID` — UUID. Body/query: нет.

Успех `200` — подробная попытка. `answers` содержит сохранённые ID:

~~~json
{
  "id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
  "quiz_question_id": "77777777-7777-4777-8777-777777777777",
  "quiz_answer_option_id": "88888888-8888-4888-8888-888888888888"
}
~~~

Ответ позволяет полностью восстановить экран после reload. Если deadline уже
наступил, статус не будет означать, что ответы ещё принимаются. Правильность
вариантов не раскрывается. Ошибки: `400`, `401`, `404`, `500`.
Запрос безопасно повторять.

### PUT /v1/attempts/{attemptID}/answers/{quizQuestionID}

Login: да. CSRF: да. Оба path parameter — UUID. JSON до 1 MiB:

~~~json
{
  "option_id": "88888888-8888-4888-8888-888888888888"
}
~~~

Option обязан принадлежать этому quiz question, вопрос — quiz попытки, а
попытка — пользователю и иметь `in_progress` до deadline.

Успех `200`:

~~~json
{
  "quiz_question_id": "77777777-7777-4777-8777-777777777777",
  "option_id": "88888888-8888-4888-8888-888888888888",
  "updated_at": "2026-11-02T10:35:00Z"
}
~~~

Правильность в ответе отсутствует. Ошибки: `400` UUID/JSON, `401`, `403`,
`404` чужая связь/ресурс, `409` попытка закрыта или deadline наступил,
`413`, `415`, `500`.

Повтор того же выбора безопасен и оставляет одну строку ответа; `updated_at`
может обновиться. Другой `option_id` заменяет выбор в той же строке.

### POST /v1/attempts/{attemptID}/finish

Login: да. CSRF: да. `attemptID` — UUID. Body/query: нет. Клиент не отправляет
score, `is_correct` или время завершения.

Успех `200` — Result object:

~~~json
{
  "attempt_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "quiz_id": "66666666-6666-4666-8666-666666666666",
  "status": "completed",
  "total_questions": 1,
  "answered_questions": 1,
  "correct_answers": 1,
  "wrong_answers": 0,
  "unanswered_questions": 0,
  "percentage": 100,
  "started_at": "2026-11-02T10:30:00Z",
  "finished_at": "2026-11-02T10:40:00Z",
  "correct_answers_list": [
    {
      "id": "77777777-7777-4777-8777-777777777777",
      "text": "Столица Франции?",
      "position": 1,
      "selected_option_id": "88888888-8888-4888-8888-888888888888",
      "is_correct": true,
      "options": [
        {
          "id": "88888888-8888-4888-8888-888888888888",
          "quiz_question_id": "",
          "text": "Париж",
          "position": 1,
          "is_correct": true,
          "is_selected": true
        }
      ]
    }
  ],
  "wrong_answers_list": [],
  "unanswered_questions_list": []
}
~~~

Если deadline наступил, status будет `expired`, иначе `completed`.
`percentage = correct_answers / total_questions * 100`; при нуле вопросов
сервер возвращает 0. Полное правило зафиксировано в [decisions.md](decisions.md).
У пропущенного вопроса `selected_option_id` и `is_correct` равны `null`.
Текущий результат возвращает `options[].quiz_question_id` как пустую строку.
Для связи с вопросом используй `id` родительского объекта.

Ошибки: `400`, `401`, `403`, `404`, `500`. Finish идемпотентен по результату:
повтор не создаёт новую попытку, не переносит `finished_at` и возвращает
сохранённый статус с тем же результатом.

### GET /v1/attempts/{attemptID}/result

Login: да. CSRF: нет. `attemptID` — UUID. Body/query: нет.

Успех `200` — тот же Result object, что Finish. Для уже закрытой попытки
возвращается стабильный результат. Для просроченной `in_progress` попытки
сервер сначала фиксирует `expired`, затем возвращает результат.

Ошибки: `400`, `401`, `404`, `409` если попытка ещё активна и deadline не
наступил, `500`. Запрос безопасно повторять. Это основной endpoint для
повторного открытия страницы результата.

### GET /v1/quizzes/{quizID}/attempts

Login: да. CSRF: нет. `quizID` — UUID. Query: `limit`, `offset`.

~~~http
GET /v1/quizzes/66666666-6666-4666-8666-666666666666/attempts?limit=20&offset=0
~~~

Успех `200` — массив попыток.

~~~json
[
  {
    "attempt_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "quiz_id": "66666666-6666-4666-8666-666666666666",
    "status": "completed",
    "started_at": "2026-11-02T10:30:00Z",
    "deadline_at": "2026-11-02T11:00:00Z",
    "finished_at": "2026-11-02T10:40:00Z",
    "correct_answers": 1,
    "gradable_count": 1,
    "percentage": 100
  }
]
~~~

Пустая страница возвращает `[]`. Для активной попытки `percentage` равен `null`.
Истёкший deadline учитывается при чтении истории. Показывай действие продолжения
только для `in_progress`.

Ошибки: `400` UUID/query, `401`, `404`, `500`. Запрос безопасно повторять.

## Служебные endpoints

### GET /health

Login/CSRF: нет. Body/path/query: нет.

Успех `200`:

~~~json
{"status":"ok"}
~~~

Проверяет, что HTTP-процесс работает. Запрос безопасно повторять.

### GET /ready

Login/CSRF: нет. Body/path/query: нет.

Успех `200`:

~~~json
{"status":"ready"}
~~~

Если PostgreSQL недоступен, `503`:

~~~json
{"error":"not ready"}
~~~

Запрос безопасно повторять.

## Лимиты и rate limiting

| Ресурс | Лимит |
| --- | --- |
| Login/register/verify/resend JSON | 16 KiB |
| Обычный JSON | 1 MiB |
| Import confirmation JSON | 10 MiB |
| DOCX-файл | 10 MiB |
| Multipart upload | 11 MiB |
| Извлекаемый DOCX XML | 20 MiB |
| Одновременные preview/imports | 4 на процесс |
| Login | 10 запросов/минуту/IP |
| Register | 5 запросов/час/IP |
| Verify email | 60 запросов/час/IP |
| Resend verification | 10 запросов/час/IP |
| Google Start + Callback | общий лимит 30 запросов/минуту/IP |
| Выдача verification token | 1 за 60 секунд, не более 5/час/аккаунт |
| Preview + import | 10 запросов/минуту/пользователя |

Rate limiter текущей версии хранится в памяти одного backend. Перезапуск
обнуляет IP-счётчики. Лимит выдачи verification token хранится в БД.

## Insomnia collection

Файл [Question-To-Quiz.insomnia.json](insomnia/Question-To-Quiz.insomnia.json)
использует импортируемый Insomnia v4 JSON. Современная Insomnia продолжает
принимать этот формат.

После импорта:

1. Выбрать sub-environment `Local` или `Staging`.
2. Для Staging заменить `.invalid` URL на настоящий HTTPS origin.
3. Ввести `email` и `password` только в локальной среде Insomnia.
4. Выполнить Register, открыть письмо и подтвердить email. Затем Login и CSRF.
   Для теста через Insomnia можно вручную скопировать token из фрагмента ссылки
   в локальную переменную `verification_token` и выполнить Verify Email.
5. Затем выполнить основной путь: создать банк, вопрос, quiz, attempt,
   сохранить answer, Finish, Get Result.
6. Для Preview вручную выбрать DOCX в multipart field `file`.

Collection содержит пустой cookie jar и пустые значения
`password`, `csrf_token`, `verification_token`. Resource ID подставляются
из ответов во время выполнения. После Login Insomnia хранит cookie в runtime.
After-response scripts сохраняют полученные resource ID и CSRF token в выбранный
environment для следующих запросов.

Перед повторным экспортом из Insomnia очистить cookie jar, `password`,
`csrf_token`, `verification_token` и runtime ID. Пометка confidential не делает экспорт безопасным:
значения confidential variables могут попасть в экспорт в открытом виде.

Особое поведение:

- повторный Import добавляет ещё один набор вопросов;
- повторный Start возвращает текущую активную попытку, а после закрытия создаёт новую;
- повторный Finish возвращает результат той же закрытой попытки;
- SaveAnswer делает UPSERT одного выбора для одного вопроса.

## Безопасность и эксплуатация

Настройка писем, Gmail App Password, локального inbox и миграции:
[email-verification.md](email-verification.md). Миграция сохраняет доступ
старым аккаунтам: их `email_verified_at` заполняется датой перехода на новую
версию. Это политика совместимости, а не доказательство проверки их почты.

Production требует `APP_ENV=production` и явный
`PUBLIC_ORIGIN=https://ваш-домен`. Cookie и CSRF token не логируются. Ответы
имеют `Cache-Control: no-store`, `X-Content-Type-Options: nosniff` и
`X-Request-ID`.

HTTP request deadline — 30 секунд. `/ready` проверяет PostgreSQL с timeout
2 секунды. Реальные секреты задаются окружением; `.env.example` содержит только
placeholders.

Справочные материалы:

- [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [OWASP CSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [Go net/http](https://pkg.go.dev/net/http)
- [Insomnia import/export](https://developer.konghq.com/insomnia/import-export/)
- [Insomnia environments](https://developer.konghq.com/insomnia/environments/)
