<div align="center"><img width="500" src="../assets/dle.svg" border="0" /></div>

<div align="center"><h1 align="center">Database Lab Engine (DLE)</h1></div>

<div align="center">
  <a href="https://twitter.com/intent/tweet?via=Database_Lab&url=https://github.com/postgres-ai/database-lab-engine/&text=Thin%20@PostgreSQL%20clones%20–%20DLE%20provides%20blazing%20fast%20database%20cloning%20to%20build%20powerful%20development,%20test,%20QA,%20staging%20environments.">
    <img src="https://img.shields.io/twitter/url/https/github.com/postgres-ai/database-lab-engine.svg?style=for-the-badge" alt="twitter">
  </a>
</div>

<div align="center">
  <strong>:zap: Молниеносное клонирование баз данных PostgreSQL :elephant:</strong><br>
  Тонкие клоны для создания dev / test / QA / staging сред.<br>
  <sub>Доступно для любых PostgreSQL, включая AWS RDS, GCP CloudSQL, Heroku, Digital Ocean и серверы, администрируемых пользователем</sub>
</div>

<br />

<div align="center">
  <a href="https://postgres.ai" target="blank"><img src="https://img.shields.io/badge/Postgres-AI-orange.svg?style=flat" /></a> <a href="https://github.com/postgres-ai/database-lab-engine/releases/latest"><img src="https://img.shields.io/github/v/release/postgres-ai/database-lab-engine?color=orange&label=Database+Lab&logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAYCAYAAACWTY9zAAAACXBIWXMAAAsTAAALEwEAmpwYAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAPYSURBVHgBrVc9SCNBFH7JpVCrjdpotVgFES9qp8LdgaXNFWLnJY2lsVC0zIGKQeEujRw2508lNndqISKaA38a/4Io/qBGQc2B6IKgImLufYPj7W42Jsb9YNidb2ffvHnzzZsZB1mgra3to9Pp9Docjvdc9XJR3G63qm9zdXUV44fGJZZIJKKPj4+R/v7+CNkEh3wJBoPKzc1NIC8vr7WoqEgpLS2l4uJiYodEscLd3R2dnZ2Jcnh4SNvb23ByiG2E2R6cpo6Oju/s9EZfX9+Q/C8F95O5P5ITjnV2dqq5ubnz1dXVam1tLeXk5FA24CjS6uoqLS4uxtjpT729vbGLi4ujubk5lflf3IcfDuu4CHOfJbe8vKwuLCwITno7f3p6mrALBwcHCdiEba4egYP97u7uYDru8vIy0dPT8835NFg1Pz+f7MLT1Kt6DrIoKyv7ko7Dvx6Pxycdo3A4LKbirYDWRkdHLb/t7u5mxO3t7SkuWWlubhYGoa+qqiriBSBGlAkwoK2tLYhf1Ovr62lwcNDwfXJykgoLCzPiELVnx1BpaWkRK2xtbU2IGA3Bw1kWpMGZ29tb0jRNPNGmpKSE6urqxFOPgYEBcrlcwtmVlZWMOF48/x2TQJT0kZIpwQzpbKpUIuHz+YjTh4FrbGykgoKCFzmX3gGrNAHOHIXXwOwUYHbKinsWP+YWzr0VsDE+Pp7EQxZmoafisIAMGoNgkfFl1n8NMN0QP7RZU1Nj+IaOZmdnDUJ/iTOIH8LFasTHqakp0ZHUG6bTrCUpfk6I4h+0w4ACgYBoDxsAbzFUUVFBTU1NNDMzkxGH2TOIH53DORQZBdm5Ocehc6SUyspKQnJOtY21t7dnxSWtSj3MK/StQJQz4aDTZ/Fjbu2ClS1EfGdnJ4k7OTlJ4jBTLj2B1YRpzDY9SPHqp5WPUrS0tCQ64z3QwKG9FL+eM4i/oaFBkHzsoJGREeFcOvGfn5+LJ/7DO9rI7M9HKdFubGyMysvLBT8xMWHgsA1acQiQQWMwKKOFzuQBEOI35zg4gcyvKArhDCcHYIbf78+KSyl+vZN24f7+XjNzVuJHOyn+GCJjF5721pieQ+Ll8lvPoc/19fUkbnNzc1hEjC8dfj7yzHPGViH+dBtzKmC6oVEcrWETHJ+tKBqNwqlwKBQKWnCtVtw7kGxM83q9w8fHx3/ZqIdHrFxfX9PDw4PQEY4jVsBKhuhxFpuenkbR9vf3Q9ze39XVFUcb3sTd8Xj8K3f2Q/6XCeew6pBX1Ee+seD69oGrChfV6vrGR3SN22zg+sbXvQ2+fETIJvwDtXvnpBGzG2wAAAAASUVORK5CYII=" alt="Latest release" /></a>

  <a href="https://gitlab.com/postgres-ai/database-lab/-/pipelines" target="blank"><img src="https://gitlab.com/postgres-ai/database-lab//badges/master/pipeline.svg" alt="CI pipeline status" /></a> <a href="https://goreportcard.com/report/gitlab.com/postgres-ai/database-lab" target="blank"><img src="https://goreportcard.com/badge/gitlab.com/postgres-ai/database-lab" alt="Go report" /></a>

  <a href="../CODE_OF_CONDUCT.md"><img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?logoColor=black&labelColor=white&color=blue" alt="Contributor Covenant" /></a> <a href="https://slack.postgres.ai" target="blank"><img src="https://img.shields.io/badge/Chat-Slack-blue.svg?logo=slack&style=flat&logoColor=black&labelColor=white&color=blue" alt="Community Slack" /></a> <a href="https://twitter.com/intent/follow?screen_name=Database_Lab" target="blank"><img src="https://img.shields.io/twitter/follow/Database_Lab.svg?style=social&maxAge=3600" alt="Twitter Follow" /></a>
</div>

<div align="center">
  <h3>
    <a href="#возможности">Возможности</a>
    <span> | </span>
    <a href="https://postgres.ai/docs">Документация</a>
    <span> | </span>
    <a href="https://postgres.ai/blog/tags/database-lab-engine">Блог</a>
    <span> | </span>
    <a href="#сообщество-и-поддержка">Сообщество и поддержка</a>
    <span> | </span>
    <a href="../CONTRIBUTING.md">Участие</a>
  </h3>
</div>

## Зачем это нужно?
- Создавайте dev-, QA-, staging-среды, основанные на полноразмерных базах данных, идентичных или приближенных к «боевым».
- Получите доступ к временным полноразмерным клонам «боевых» БД для анализа запросов SQL и оптимизации (смотрите также: [чат-бот для оптимизации SQL Joe](https://gitlab.com/postgres-ai/joe)).
- Автоматически тестируйте изменения БД в CI/CD-пайплайнах, чтобы не допускать инцидентов в продуктиве.

Например, клонирование 1-терабайтной базы данных PostgreSQL занимает около 10 секунд. При этом десятки независимых клонов могут работать на одной машине, обеспечивая разработку и тестирование без увеличения затрат на железо.

<p><img src="../assets/dle-demo-animated.gif" border="0" /></p>

Попробуйте сами прямо сейчас:

- зайдите на [Database Lab Platform](https://console.postgres.ai/), присоединитесь к организации "Demo" и тестируйте клонировани ~1-терабайтной демо базы данных или
- смотрите другое демо, DLE CE: https://nik-tf-test.aws.postgres.ai:446/instance, используйте демо-токен, чтобы зайти (это демо имеет самозаверенные сертификаты, так что игнорируйте жалобы браузера)

## Как это работает
Тонкое клонирование работает сверхбыстро, так как оно базируется на технологии [Copy-on-Write (CoW)](https://en.wikipedia.org/wiki/Copy-on-write#In_computer_storage). DLE поддерживает два варианта CoW: [ZFS](https://en.wikipedia.org/wiki/ZFS) (используется по умолчанию) и [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_(Linux)).

При работе с ZFS, DLE периодически создаёт новые снимки директории данных и поддерживает набор таких снимков, регулярно удаляя старые неиспользуемые. При создании новых клонов пользователи могут выбирать, на основе какого именно снимка создавать клон.

Узнать больше можно по следующим ссылкам:
- [Как это работает](https://postgres.ai/products/how-it-works)
- [Тестирование миграций БД](https://postgres.ai/products/database-migration-testing)
- [Оптимизация SQL с чатботом Joe](https://postgres.ai/products/joe)
- [Вопросы и ответы](https://postgres.ai/docs/questions-and-answers)

## С чего начать
- [Введение в Database Lab для любой БД на PostgreSQL](https://postgres.ai/docs/tutorials/database-lab-tutorial)
- [Введение в Database Lab для Amazon RDS](https://postgres.ai/docs/tutorials/database-lab-tutorial-amazon-rds)
- [Шаблон модуля Terraform (AWS)](https://postgres.ai/docs/how-to-guides/administration/install-database-lab-with-terraform)

## Изучение кейсов
- Qiwi: [Как Qiwi управляет данными для ускорения процесса разработки](https://postgres.ai/resources/case-studies/qiwi)
- GitLab: [Как GitLab построил итерационный процесс оптимизации SQL для снижения рисков инцидентов](https://postgres.ai/resources/case-studies/gitlab)

## Возможности
- Молниеносное клонирование БД Postgres - создание нового клона, готового к работе, всего за несколько секунд (вне зависимости от размера БД).
- Максимальное теоретическое количество снимков: 2<sup>64</sup>. ([ZFS](https://en.wikipedia.org/wiki/ZFS), вариант по умолчанию).
- Максимальный теоретический размер директории данных PostgreSQL: 256 квадриллионов зебибайт или 2<sup>128</sup> байт ([ZFS](https://en.wikipedia.org/wiki/ZFS), вариант по умолчанию).
- Поддерживаются все основные версии PostgreSQL: 9.6-14.
- Для реализации тонкого клонирования поддерживаются две технологии ([CoW](https://en.wikipedia.org/wiki/Copy-on-write)): [ZFS](https://en.wikipedia.org/wiki/ZFS) и [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_(Linux)).
- Все компоненты работают в Docker-контейнерах.
- UI для удобства ручных действий пользователя.
- API и CLI для удобства автоматизации работы со снимками и клонами DLE.
- Контейнеры с PostgreSQL по умолчанию поставляются с большим количеством популярных расширений ([docs](https://postgres.ai/docs/database-lab/supported-databases#extensions-included-by-default)).
- Поддерживается расширение контейнеров PostgreSQL ([docs](https://postgres.ai/docs/database-lab/supported-databases#how-to-add-more-extensions)).
- БД-источник может находиться где угодно (Postgres под управлением пользователя, Яндекс.Облако, AWS RDS, GCP CloudSQL, Azure, Timescale Cloud и т.д.) и не требует никаких изменений. Нет никакий требований для установки ZFS или Docker в БД-источники (продуктивная БД).
- Первоначальное получение данных может быть выполнено как на физическом (pg_basebackup или инструменты для бэкапов — такие как WAL-G, pgBackRest), так и на логическом (dump/restore напрямую из источника или восстановление из файлов, хранящихся в AWS S3) уровнях.
- Для логического режима поддерживается частичное восстановление данных (конкретные БД, таблицы).
- Для физического режима поддерживается постоянно обновляемое состояние ("sync container"), что, по сути, делает DLE репликой специального назначения.
- Для логического режима поддерживается периодическое полное обновление данных, полностью автоматизированное и контролируемое DLE. Есть возможность использовать несколько дисков, содержащих различные версии БД, так что процесс обновления не приводит к простою в работе с DLE и клонами.
- Сверхбыстрое восстановление на конкретную временную точку (Point in Time Recovery, PITR).
- Неиспользованные клоны автоматически удаляются.
- Опциональный флаг «защита от удаления» защищает клон от автоматического или ручного удаления.
- В конфигурации DLE можно настроить политику зачистки снимков.
- Неубиваемые клоны: клоны переживают рестарты DLE (включая случай с перезагрузкой машины).
- Команда "reset" может быть использована для переключения между разными версиями данных.
- Компонент DB Migration Checker собирает различные артефакты, полезные для тестирования БД в CI ([docs](https://postgres.ai/docs/db-migration-checker)).
- SSH port forwarding для API и Postgres-соединений.
- Параметры конфига Docker-контейнера могут быть специализированы в конфиге DLE.
- Квоты использования ресурсов для клонов: процессор, память (любые квоты контейнеров, поддерживаемые Docker).
- Параметры Postgres-конфига могут быть специализированы в конфиге DLE (отдельно для клонов, контейнеров "sync" и "promote").
- Monitoring: открытый `/healthz` (без авторизации), расширенный `/status` (требует авторизации), [Netdata-модуль](https://gitlab.com/postgres-ai/netdata_for_dle).

## Как поучаствовать в развитии проекта
### Поставьте проекту звёздочку
Самый простой способ поддержки - поставить проекту звезду на GitHub/GitLab:

![Поставьте звезду](../assets/star.gif)

### Укажите явно, что вы используете DLE
Пожалуйста, опубликуйте твит с упоминанием [@Database_Lab](https://twitter.com/Database_Lab) или поделитесь ссылкой на этот репозиторий в вашей любимой социальной сети.

Если вы используете DLE в работе, подумайте, где вы могли бы об этом упомянуть. Один из лучших способов упоминания - использование графики с ссылкой. Некоторые материалы можно найти в директории `./assets`. Пожалуйста, используйте их в своих документах, презентациях, интерфейсах приложений и вебсайтов, чтобы показать, что вы используете DLE.

HTML-код для светлых фонов:
<p>
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-light-background.svg" />
</p>

```html
<a href="http://databaselab.io">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-light-background.svg" />
</a>
```

Для тёмных фонов:
<p style="background-color: #bbb">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-dark-background.svg" />
</p>

```html
<a href="http://databaselab.io">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-dark-background.svg" />
</a>
```

### Предложите идею или сообщите об ошибке
Подробнее: [./CONTRIBUTING.md](../CONTRIBUTING.md).

### Участвуйте в разработке
Подробнее: [./CONTRIBUTING.md](../CONTRIBUTING.md).

### Справочники
- [Компоненты DLE](https://postgres.ai/docs/reference-guides/database-lab-engine-components)
- [Справочник по конфигурации DLE](https://postgres.ai/docs/database-lab/config-reference)
- [Справочник по DLE API](https://postgres.ai/swagger-ui/dblab/)
- [Справочник по Client CLI](https://postgres.ai/docs/database-lab/cli-reference)

### HowTo-инструкции 
- [Как установить Database Lab с Terraform на AWS](https://postgres.ai/docs/how-to-guides/administration/install-database-lab-with-terraform)
- [Как установить и инициализировать Database Lab CLI](https://postgres.ai/docs/guides/cli/cli-install-init)
- [Как управлять DLE](https://postgres.ai/docs/how-to-guides/administration)
- [Как работать с клонами](https://postgres.ai/docs/how-to-guides/cloning)

Вы можете найти больше в [секции "How-to guides"](https://postgres.ai/docs/how-to-guides) документации.

### Разное
- [Docker-образы DLE](https://hub.docker.com/r/postgresai/dblab-server)
- [Extended Docker images for PostgreSQL (с огромным количеством расширений)](https://hub.docker.com/r/postgresai/extended-postgres)
- [Чатбот для оптимизации SQL (чатбот Joe)](https://postgres.ai/docs/joe-bot)
- [DB Migration Checker](https://postgres.ai/docs/db-migration-checker)

## Лицензия
Код DLE распространяется под лицензией, одобренной OSI: GNU Affero General Public License version 3 (AGPLv3).

Свяжитесь с командой Postgres.ai, если вам нужна коммерческая лицензия, которая не содержит предложений GPL, а также, если вам нужна поддержка: [Контактная страница](https://postgres.ai/contact).

## Сообщество и Поддержка
- ["Кодекс поведения сообщества Database Lab Engine"](../CODE_OF_CONDUCT.md)
- Где получить помощь: [Контактная страница](https://postgres.ai/contact)
- [Сообщество в Телеграм (русский язык)](https://t.me/databaselabru)
- [Сообщество в Slack](https://slack.postgres.ai)
- Если вам надо сообщить о проблеме безопасности, следуйте инструкциям в документе ["SECURITY.md"](../SECURITY.md).

[![Кодекс поведения](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?color=blue)](../CODE_OF_CONDUCT.md)


<!--
## Переводы
- ...
-->
