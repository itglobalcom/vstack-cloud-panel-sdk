# vstack-cloud-panel-sdk

Go SDK к Public API панели (контракт `public-api`, publisher `cloudmng` — см. `workspace.yaml`
корня workspace). Потребитель SDK внутри workspace — `terraform-provider-vcp`.

Module `github.com/itglobalcom/vstack-cloud-panel-sdk`, Go по `go.mod` (сейчас 1.25.0).
Зависимости: stdlib + `golang.org/x/crypto`. Новые внешние зависимости — не добавлять без
явного запроса. GitHub — read-only зеркало, разработка во внутреннем GitLab (`CONTRIBUTING.md`).

## Структура

- корень — `package sdk`, **один файл на домен** (`server.go`, `network.go`, `dns.go`,
  `vmware.go`, …). Инфраструктура: `client.go`, `config.go`, `retry.go`, `errors.go`,
  `logger.go`, `task.go`, `doc.go`. Вложенных пакетов нет.
- `entities/` — модели, файл зеркалит доменный файл корня (`vmware.go` ↔ `entities/vmware.go`).
- `examples/` — `package main`, файл на ресурс + диспетчер `examples/main.go` (`make example RESOURCE=…`).
- тесты — в корне, `<домен>_internal_test.go`, `package sdk`.

## Клиент

- Контекст — всегда первый параметр `ctx context.Context`; клиент сам контекст не создаёт.
- Запрос идёт только через `c.newRequest(ctx, method, path, body)` + `c.doJSON(req, &resp)`.
  Свой `http.Do`, свои заголовки и свой разбор статусов в доменных файлах — не писать.
- Путь передаётся **без** `/api/v1` — префикс добавляет `c.buildURL`.
- Пути — через const в начале доменного файла + приватный билдер (`buildVMwarePath`,
  `buildNetworkPath`). Query — `url.Values`, приклеивается билдером.
- Аутентификация — заголовок `X-API-KEY` (+`User-Agent`), ставится в `headerTransport`
  (`client.go`), не в доменных методах.
- Именование: `GetX` (одиночный), `GetXList` (список верхнего уровня), `GetXs` (вложенная
  коллекция), `CreateX/UpdateX/PatchX/DeleteX/RenameX`, `PowerOn|PowerOff|Shutdown|Reboot|ResetServer`.
- Операция с фоновой задачей: базовый метод возвращает `*TaskID`, к нему парный
  `…AndWait` через `c.waitTaskCompletion` — оба, если операция асинхронная.
- Типы ответов — в доменном файле, групповой блок `type (…)` под `// Response types`;
  они разворачивают обёртку API (`{"isolated_network": {…}}`).
- Аргументы валидируются **до** построения запроса (`if id == "" { return nil, fmt.Errorf("… is required") }`),
  для request-структур — `req.Validate()`.
- Каждая ошибка оборачивается `%w` с контекстом: `fmt.Errorf("failed to get network %s: %w", id, err)`.

## Entities

- JSON-теги — `snake_case`, как в контракте publisher'а; опциональные с `,omitempty`.
- Go-имена идиоматичные, с сохранением аббревиатур: `ID`, `GPUSupported`, `NICHotRemove`, `MinGB`.
- В одном файле: сущность + `CreateXRequest`/`UpdateXRequest` + доменные typed-константы.
- `Validate() error` — на указателе request-структуры.
- Doc-комментарий над каждым экспортируемым типом; где важны единицы измерения или
  неочевидная семантика контракта — комментарий у поля.

## Ошибки

- `*RequestError` (`errors.go`): статус ≥400 → `parseAPIError` (формат
  `{"errors":[{"code":…,"message":…}]}`, есть legacy-fallback).
- Коды API — именованные константы `APICode*` с комментарием, воспроизводящим текст API.
- Проверка кода — предикатами `IsNotFound/IsAlreadyExists/IsConflict/IsInvalidLocation/HasAPICode`
  (внутри `errors.As`, работают через `%w`). Вручную сравнивать `StatusCode`/коды в
  вызывающем коде — не надо; нужен новый случай — добавь предикат и константу.
- `ErrNotFound` — sentinel для «API ответил 200, но объекта нет».

## Тесты

- Только stdlib `testing`. Нет testify, моков, golden-файлов — так и оставить.
- Сетевой сценарий проверяется на стабе `net/http/httptest` через готовый хелпер
  `newTestClient(t, handler)` (`gateway_internal_test.go`): он направляет клиент на стаб
  и обнуляет ожидания ретраев. Свой `httptest.NewServer` в доменных тестах не поднимать.
- Файл `<домен>_internal_test.go`, `package sdk` (тестируются и приватные функции).
- Паттерны: map-driven `cases := map[string]string{…}` для чистых функций; `t.Run` для
  сценариев; разбор контракта — unmarshal реального JSON-литерала в `ListXResponse`
  (проверяет теги без сети); предикаты ошибок — на вручную собранном `*RequestError`.
- Формат сообщений: `t.Errorf("f(%q) = %q, want %q", …)`.
- Обязательное покрытие change-slice: билдер пути, валидация аргументов, разбор ответа
  (включая ответ без коллекции — API опускает пустые поля), новый предикат ошибки.

## CHANGELOG

`CHANGELOG.md` — часть каждого change-slice, а не работа релиза: запись идёт
в `## [Unreleased]` (Keep a Changelog: `Added`/`Changed`/`Fixed`) и описывает
поведение контракта для потребителя SDK, а не список файлов. Заголовок версии
проставляется при выпуске тега, `[Unreleased]` остаётся пустым до следующего слайса.
Незакрытый заголовок `[Unreleased]` со старой версией — расхождение с тегами,
а не вторая ожидающая версия.

## Гейты

`.github/workflows/ci.yml` — Go из `go.mod`, затем:

```
gofmt -l .        # вывод должен быть пустым
go vet ./...
go build ./...
go test -race ./...
```

Локально: `make fmt`, `make vet`, `make test` (без `-race` — race проверяет CI, перед
пушем прогоняй `go test -race ./...`). Внешних линтеров нет.

## Границы

- Контракт `public-api` определяет publisher `cloudmng`; SDK его только потребляет —
  расхождение имени/типа поля правится в SDK, а не «дополняется» своей моделью.
- Межрепозиторные зависимости описаны только в `workspace.yaml` корня — не дублировать их здесь.
- Коммиты — в ветку задачи `<TSK…>`, сообщение начинается с `TSK…`.
- Новые `.md` без явной задачи не создавать: правила — здесь, описание для людей — `README.md`,
  package-level godoc — `doc.go`.
