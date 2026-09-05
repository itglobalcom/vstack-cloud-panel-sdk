# `archi.md` — vstack-cloud-panel-sdk

> Архитектурный контракт репозитория. Кратко: границы, ограничения, контракты.
> Межрепозиторные рёбра — только в `workspace.yaml` корня workspace.

## 1) Context ID / Owner / Repository mapping

| Context ID | Owner | Repository | Code paths |
| --- | --- | --- | --- |
| ctx_go_sdk | team/arch-owner | vstack-cloud-panel-sdk | `*.go` (корень, `package sdk`), `entities/`, `examples/`, `go.mod`, `Makefile`, `.github/workflows/ci.yml` |

Репозиторий одноконтекстный: клиент, модели и примеры — один контекст. Вложенных
пакетов, кроме `entities` и `examples`, нет.

## 2) Boundaries (in/out of scope)

- In scope: типизированный Go-клиент контракта `public-api` — построение путей
  и query, аутентификация ключом, ретраи, разбор ошибок API и предикаты по кодам,
  ожидание фоновых задач, модели запросов/ответов и их `Validate()`, runnable-примеры
  на каждый раздел.
- Out of scope: доменные правила и валидация услуг (владелец — publisher `cloudmng`) —
  SDK повторяет контракт, а не дополняет его своей моделью; остальные API монолита
  (`partner-api`, `referral-api`, Admin API v3); генерация клиента из OpenAPI — код
  рукописный; поведение потребителей SDK.
- Primary L1 context: `ctx_go_sdk`; Affected L1 contexts: нет.

## 3) Layering constraints

- Слои: `entities` → `sdk` (корень) → `examples`. Внутри `sdk` инфраструктура
  (`client.go`, `config.go`, `retry.go`, `errors.go`, `logger.go`) обслуживает
  доменные файлы (`server.go`, `dns.go`, `vmware_*.go`, …), файл на домен.
- Разрешено: доменный файл строит запрос только через `c.newRequest` + `c.doJSON`,
  путь собирает приватным билдером (`buildVMwarePath`, `buildNetworkPath`) поверх
  const'ов своего файла; типы ответов, разворачивающие обёртку API, живут в том же
  доменном файле.
- Запрещено: импорт `sdk` из `entities` (обратная зависимость слоёв); свой
  `http.Do`, свои заголовки и свой разбор статусов в доменных файлах; ручное
  сравнение `StatusCode` и кодов ошибок у вызывающего вместо предикатов
  (`IsNotFound`, `HasAPICode`, …); префикс `/api/v1` в пути (его добавляет
  `c.buildURL`); новые вложенные пакеты; новые внешние зависимости.

## 4) External contracts (API/events)

| Contract | Type | Direction | Owner | Compatibility |
| --- | --- | --- | --- | --- |
| `public-api` — REST монолита | API | in | cloudmng | consumer: SDK следует за publisher'ом и своего контракта не определяет; расхождение имени/типа поля правится в SDK. `PUT /vmware/networks/{id}/edge/bandwidth` применяет значение начиная с релиза платформы, в котором выехал фикс маршрута (TSK0003840); на более старом развёртывании задача завершается, а полоса остаётся прежней |
| Go-модуль `github.com/itglobalcom/vstack-cloud-panel-sdk` — экспортированная поверхность пакетов `sdk` и `entities` | Go module | out | ctx_go_sdk | SemVer Go-модулей: на `v1` допустимы только аддитивные релизы. Удаление или переименование экспортированной декларации требует `v2.0.0` **и суффикса `/v2` в пути модуля**, то есть правки import-путей у всех потребителей, включая внешних. Устаревшее помечается `// Deprecated:` и остаётся опубликованным до мажора |

Публикуется тегом `vX.Y.Z`; GitHub — read-only зеркало, разработка во внутреннем
GitLab (`CONTRIBUTING.md`).

## 5) Quality gates and testing

- Tests: разбор контракта (unmarshal реального JSON-литерала в response-тип —
  проверяет теги без сети) — Required на каждый новый/изменённый response-тип;
  билдер пути и query — Required; валидация аргументов и `Validate()`
  request-структур — Required; новый предикат ошибки — Required. Сценарии
  с запросом идут через стаб-сервер `httptest` (хелпер `newTestClient`,
  `gateway_internal_test.go`). Тест лежит у владельца артефакта: корневой
  `<домен>_internal_test.go` (`package sdk`) — для клиента, `entities/*_test.go` —
  для моделей и их `Validate()`.
- Gates (`.github/workflows/ci.yml`, версия Go — из `go.mod`): `gofmt -l .` (вывод
  пустой), `go vet ./...`, `go build ./...`, `go test -race ./...`. Локально —
  `make fmt`, `make vet`, `make test`; `-race` перед пушем прогонять вручную.
  Внешних линтеров нет.
- Done: гейты зелёные; у каждого нового метода есть пример в `examples/` и строка
  в `README.md`; асинхронная операция имеет пару `…AndWait`; изменение поверхности
  отражено в `CHANGELOG.md` в разделе, соответствующем его совместимости.

## 6) Known legacy deviations

| Deviation | Impact/Risk | Mitigation plan |
| --- | --- | --- |
| Идентификаторы ресурсов в сигнатурах — примитивы: `string` в разделе vStack (`GetServer(ctx, serverID string)`), `int` в разделе VMware (`GetVmwareServer(ctx, serverID int)`). Типизированы только ссылки на задачу — `TaskID`, `VmwareTaskID` | нарушение GR-02: соседние `int`-аргументы (`serverID`, `volumeID`, `locationID`) можно переставить местами, компилятор этого не заметит | завести доменные типы идентификаторов нельзя аддитивно — смена типа параметра ломает компиляцию у потребителей и требует `/v2`. Переносится в план мажора; до тех пор новые методы держат порядок аргументов «владелец → вложенный ресурс», как у существующих |
| Семейство `GetVMware*` каталога (`GetVMwareLocations`, `GetVMwareImages`, `GetVMwareGPUModels`, `GetVMwareDiskTypes`, `GetVMwareStorageProfiles`) с их `ListVMware*Response`, `entities.VMwareDiskType` и `entities.VMwareStorageProfile` | описывают контракт, которого нет: `/vmware/disk-types` и `/vmware/storage-profiles` опубликованы только под префиксом AdminV2, через Public API отвечают 404; поля `VMwareDiskType` (`ID`, `MinGB`, `MaxGB`, `StepGB`, `StartValueGB`) не имеют соответствия на проводе и декодируются в нули | помечено `// Deprecated:` с указанием замены (`GetVmware*List`, `VmwareLocation.DiskTypes`) и факта 404; удаление — только в `v2.0.0`, потому что на `v1` оно требует суффикса `/v2` в пути модуля при нулевой выгоде: рабочих вызовов у этих методов нет |
| `entities.VMwareLocation` объявлен алиасом `VmwareLocation` и потому несёт больше полей, чем прежняя структура | composite literal без имён полей (`entities.VMwareLocation{2, "ds-msk", true}`) перестаёт компилироваться; литералы с именами полей и чтение полей не затронуты | принято: расширение структуры — аддитивное изменение по практике Go (так растут структуры stdlib); альтернатива — дублирующий тип с той же ролью |
| Раздел Kubernetes контракта `public-api` в SDK не покрыт | паритет с контрактом неполный: кластерами управляют панель, CLI и прямые вызовы API | поддержать разделом целиком, когда сервис `feature-k8s` выйдет из режима поддержки; «частичного» покрытия раздела не заводить |

## 7) Change policy and required reviews

- Policy: релиз на `v1` — аддитивный. Экспортированная декларация не удаляется
  и не переименовывается: устаревшее получает `// Deprecated:` с заменой и остаётся
  опубликованным. Поведение метода правится (ошибочный параметр запроса, неверный
  тег), поверхность — нет. Новый метод следует именованию репозитория
  (`GetX`/`GetXList`/`GetXs`, `CreateX`/`UpdateX`/`DeleteX`), асинхронный — с парой
  `…AndWait`. Новые внешние зависимости не добавляются.
- Required reviews: владелец контекста; интеграции — при добавлении раздела
  контракта `public-api` в SDK или изменении сигнатуры существующего метода.
- Escalation triggers: любое удаление, переименование или смена типа
  в экспортированной поверхности (мажор + `/v2`); изменение семантики уже
  опубликованного метода; обращение к API, отличному от `public-api`; HTTP-вызов
  в обход `c.newRequest`/`c.doJSON`; новая внешняя зависимость.
