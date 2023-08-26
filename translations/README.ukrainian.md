<div align="center">
  <img width="500" src="../assets/dle.svg" border="0" />
  <sub><br /><a href="./README.german.md">Deutsch</a> | <a href="./README.portuguese-br.md">Português (BR)</a> | <a href="./README.russian.md">Русский</a> | <a href="./README.spanish.md">Español</a> | <a href="./README.ukrainian.md">Українська</a></sub>
</div>

<br />

<div align="center"><h1 align="center">Database Lab Engine (DLE)</h1></div>

<div align="center">
  <a href="https://twitter.com/intent/tweet?via=Database_Lab&url=https://github.com/postgres-ai/database-lab-engine/&text=Thin%20@PostgreSQL%20clones%20–%20DLE%20provides%20blazing%20fast%20database%20cloning%20to%20build%20powerful%20development,%20test,%20QA,%20staging%20environments.">
    <img src="https://img.shields.io/twitter/url/https/github.com/postgres-ai/database-lab-engine.svg?style=for-the-badge" alt="twitter">
  </a>
</div>

<div align="center">
  <strong>:zap: Блискавичне клонування баз даних PostgreSQL :elephant:</strong><br>
  Тонкі клони для створення dev/test/QA/staging середовищ.<br>
  <sub>Доступно для будь-яких PostgreSQL, включаючи AWS RDS, GCP CloudSQL, Heroku, Digital Ocean та серверів, що адмініструються користувачем</sub>
</div>

<br />

<div align="center">
  <a href="https://postgres.ai" target="blank"><img src="https://img.shields.io/badge/Postgres-AI-orange.svg?style=flat" /></a> <a href="https://github.com/postgres-ai/database-lab-engine/releases/latest"><img src="https://img.shields.io/github/v/release/postgres-ai/database-lab-engine?color=orange&label=Database+Lab&logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAYCAYAAACWTY9zAAAACXBIWXMAAAsTAAALEwEAmpwYAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAPYSURBVHgBrVc9SCNBFH7JpVCrjdpotVgFES9qp8LdgaXNFWLnJY2lsVC0zIGKQeEujRw2508lNndqISKaA38a/4Io/qBGQc2B6IKgImLufYPj7W42Jsb9YNidb2ffvHnzzZsZB1mgra3to9Pp9Docjvdc9XJR3G63qm9zdXUV44fGJZZIJKKPj4+R/v7+CNkEh3wJBoPKzc1NIC8vr7WoqEgpLS2l4uJiYodEscLd3R2dnZ2Jcnh4SNvb23ByiG2E2R6cpo6Oju/s9EZfX9+Q/C8F95O5P5ITjnV2dqq5ubnz1dXVam1tLeXk5FA24CjS6uoqLS4uxtjpT729vbGLi4ujubk5lflf3IcfDuu4CHOfJbe8vKwuLCwITno7f3p6mrALBwcHCdiEba4egYP97u7uYDru8vIy0dPT8835NFg1Pz+f7MLT1Kt6DrIoKyv7ko7Dvx6Pxycdo3A4LKbirYDWRkdHLb/t7u5mxO3t7SkuWWlubhYGoa+qqiriBSBGlAkwoK2tLYhf1Ovr62lwcNDwfXJykgoLCzPiELVnx1BpaWkRK2xtbU2IGA3Bw1kWpMGZ29tb0jRNPNGmpKSE6urqxFOPgYEBcrlcwtmVlZWMOF48/x2TQJT0kZIpwQzpbKpUIuHz+YjTh4FrbGykgoKCFzmX3gGrNAHOHIXXwOwUYHbKinsWP+YWzr0VsDE+Pp7EQxZmoafisIAMGoNgkfFl1n8NMN0QP7RZU1Nj+IaOZmdnDUJ/iTOIH8LFasTHqakp0ZHUG6bTrCUpfk6I4h+0w4ACgYBoDxsAbzFUUVFBTU1NNDMzkxGH2TOIH53DORQZBdm5Ocehc6SUyspKQnJOtY21t7dnxSWtSj3MK/StQJQz4aDTZ/Fjbu2ClS1EfGdnJ4k7OTlJ4jBTLj2B1YRpzDY9SPHqp5WPUrS0tCQ64z3QwKG9FL+eM4i/oaFBkHzsoJGREeFcOvGfn5+LJ/7DO9rI7M9HKdFubGyMysvLBT8xMWHgsA1acQiQQWMwKKOFzuQBEOI35zg4gcyvKArhDCcHYIbf78+KSyl+vZN24f7+XjNzVuJHOyn+GCJjF5721pieQ+Ll8lvPoc/19fUkbnNzc1hEjC8dfj7yzHPGViH+dBtzKmC6oVEcrWETHJ+tKBqNwqlwKBQKWnCtVtw7kGxM83q9w8fHx3/ZqIdHrFxfX9PDw4PQEY4jVsBKhuhxFpuenkbR9vf3Q9ze39XVFUcb3sTd8Xj8K3f2Q/6XCeew6pBX1Ee+seD69oGrChfV6vrGR3SN22zg+sbXvQ2+fETIJvwDtXvnpBGzG2wAAAAASUVORK5CYII=" alt="Latest release" /></a>

  <a href="https://gitlab.com/postgres-ai/database-lab/-/pipelines" target="blank"><img src="https://gitlab.com/postgres-ai/database-lab//badges/master/pipeline.svg" alt="CI pipeline status" /></a> <a href="https://goreportcard.com/report/gitlab.com/postgres-ai/database-lab" target="blank"><img src="https://goreportcard.com/badge/gitlab.com/postgres-ai/database-lab" alt="Go report" /></a>

  <a href="../CODE_OF_CONDUCT.md"><img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?logoColor=black&labelColor=white&color=blue" alt="Contributor Covenant" /></a> <a href="https://slack.postgres.ai" target="blank"><img src="https://img.shields.io/badge/Chat-Slack-blue.svg?logo=slack&style=flat&logoColor=black&labelColor=white&color=blue" alt="Community Slack" /></a> <a href="https://twitter.com/intent/follow?screen_name=Database_Lab" target="blank"><img src="https://img.shields.io/twitter/follow/Database_Lab.svg?style=social&maxAge=3600" alt="Twitter Follow" /></a>
</div>

<div align="center">
  <h3>
    <a href="#можливості">Можливості</a>
    <span> | </span>
    <a href="https://postgres.ai/docs">Документація</a>
    <span> | </span>
    <a href="https://postgres.ai/blog/tags/database-lab-engine">Блог</a>
    <span> | </span>
    <a href="#спільнота-та-підтримка">Спільнота та підтримка</a>
    <span> | </span>
    <a href="../CONTRIBUTING.md">Участь</a>
  </h3>
</div>

## Навіщо це потрібно?
- Створюйте dev-, QA-, staging-середовища, засновані на повнорозмірних базах даних, ідентичних або наближених до «бойових».
- Отримайте доступ до тимчасових повнорозмірних клонів «бойової» БД для аналізу запитів SQL та оптимізації (дивіться також: [чат-бот для оптимізації SQL Joe](https://gitlab.com/postgres-ai/joe)).
- Автоматично тестуйте зміни БД у CI/CD-пайплайнах, щоб не допускати інцидентів у продуктиві.

Наприклад, клонування 1-терабайтної бази даних PostgreSQL займає близько 10 секунд. При цьому десятки незалежних клонів можуть працювати на одній машині, забезпечуючи розробку та тестування без збільшення витрат на залізо.

<p><img src="../assets/dle-demo-animated.gif" border="0" /></p>

Спробуйте самі:

- зайдіть на [Database Lab Platform](https://console.postgres.ai/), приєднайтесь до організації "Demo" і тестуйте клонування ~1-терабайтної демо бази даних або
- дивіться інше демо, DLE CE: https://nik-tf-test.aws.postgres.ai:446/instance, використовуйте демо-токен, щоб зайти (це демо має самозавірені сертифікати, так що ігноруйте скарги браузера)

## Як це працює
Тонке клонування працює надшвидко, оскільки воно базується на технології [Copy-on-Write (CoW)] (https://en.wikipedia.org/wiki/Copy-on-write#In_computer_storage). DLE підтримує два варіанти CoW: [ZFS](https://en.wikipedia.org/wiki/ZFS) (використовується за замовчуванням) та [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_(Linux)).

При роботі з ZFS DLE періодично створює нові знімки директорії даних і підтримує набір таких знімків, періодично зачищаючи старі невикористовувані. При створенні нових клонів користувачі можуть вибрати, на основі якого саме знімка створювати клон.

Дізнатися більше можна за наступними посиланнями:
- [Як це працює](https://postgres.ai/products/how-it-works)
- [Тестування міграцій БД](https://postgres.ai/products/database-migration-testing)
- [Оптимізація SQL із чатботом Joe](https://postgres.ai/products/joe)
- [Питання та відповіді](https://postgres.ai/docs/questions-and-answers)

## З чого почати
- [Вступ до Database Lab для будь-якої БД на PostgreSQL](https://postgres.ai/docs/tutorials/database-lab-tutorial)
- [Вступ до Database Lab для Amazon RDS](https://postgres.ai/docs/tutorials/database-lab-tutorial-amazon-rds)
- [Шаблон модуля Terraform (AWS)](https://postgres.ai/docs/how-to-guides/administration/install-database-lab-with-terraform)

## Вивчення кейсів
- Qiwi: [Як Qiwi керує даними для прискорення процесу розробки](https://postgres.ai/resources/case-studies/qiwi)
- GitLab: [Як GitLab побудував ітеративний процес оптимізації SQL для зниження ризиків інцидентів](https://postgres.ai/resources/case-studies/gitlab)

## Можливості
- блискавичне клонування БД Postgres - створення нового клону, готового до роботи, всього за кілька секунд (незалежно від розміру БД).
- Максимальна теоретична кількість знімків: 2<sup>64</sup>. ([ZFS](https://en.wikipedia.org/wiki/ZFS), варіант за замовчуванням).
- Максимальний теоретичний розмір директорії даних PostgreSQL: 256 квадрильйонів зебібайт або 2<sup>128</sup> байт ([ZFS](https://en.wikipedia.org/wiki/ZFS), варіант за замовчуванням).
- Підтримуються усі основні версії PostgreSQL: 9.6-14.
- Для реалізації тонкого клонування підтримуються дві технології ([CoW](https://en.wikipedia.org/wiki/Copy-on-write)): [ZFS](https://en.wikipedia.org/wiki/ZFS ) та [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_(Linux)).
- Усі компоненти працюють у Docker-контейнерах.
- UI для зручності ручних дій користувача.
- API та CLI для зручності автоматизації роботи зі знімками та клонами DLE.
- За замовчуванням контейнери PostgreSQL включають безліч популярних розширень ([docs](https://postgres.ai/docs/database-lab/supported-databases#extensions-included-by-default)).
- Контейнери PostgreSQL можуть бути кастомізовані ([docs](https://postgres.ai/docs/database-lab/supported-databases#how-to-add-more-extensions)).
- БД-джерело може бути де завгодно (Postgres під управлінням користувача, Yandex.Cloud, AWS RDS, GCP CloudSQL, Azure, Timescale Cloud і т.д.) і не вимагає жодних змін. Немає жодних вимог для встановлення ZFS або Docker у БД-джерела (продуктивна БД).
- Початкове отримання даних може бути виконане як на фізичному (pg_basebackup або інструменти для бекапів - такі як WAL-G, pgBackRest), так і на логічному (dump/restore безпосередньо з джерела або відновлення з файлів, що зберігаються в AWS S3) рівнях.
– Для логічного режиму підтримується часткове відновлення даних (конкретні БД, таблиці).
- Для фізичного режиму підтримується постійно оновлюваний стан ("sync container"), що по суті робить DLE спеціалізованою реплікою.
- Для логічного режиму підтримується періодичне повне оновлення даних, повністю автоматизоване та контрольоване DLE. Є можливість використовувати кілька дисків, що містять різні версії БД, тому процес оновлення не призводить до простою в роботі з DLE і клонами.
- Надшвидке відновлення на певний момент у часі (Point in Time Recovery, PITR).
- Невикористані клони автоматично видаляються.
- Опція "Захист від видалення" захищає клон від автоматичного або ручного видалення.
- У конфігурації DLE можна настроїти політику зачистки знімків.
- Невбивні клони: клони переживають рестарти DLE (включаючи випадок із перезавантаженням машини).
- Команда "reset" може бути використана для перемикання між різними версіями даних.
- Компонент DB Migration Cheecker збирає різні артефакти, корисні для тестування БД у CI ([docs](https://postgres.ai/docs/db-migration-checker)).
- SSH port forwarding для API та Postgres-з'єднань.
- Параметри конфігурації Docker-контейнера можуть бути спеціалізовані в конфігурацію DLE.
- Квоти використання ресурсів для клонів: процесор, пам'ять (будь-які квоти контейнерів, що підтримуються Docker).
- Параметри Postgres конфігурації можуть бути спеціалізовані в конфігурації DLE (окремо для клонів, контейнерів "sync" і "promote").
- Monitoring: відкритий `/healthz` (без авторизації), розширений `/status` (вимагає авторизації), [Netdata-модуль](https://gitlab.com/postgres-ai/netdata_for_dle).

## Як взяти участь у розвитку проекту
### Поставте проекту зірочку
Найпростіший спосіб підтримки - поставити проекту зірку на GitHub/GitLab:

![Поставте зірку](../assets/star.gif)

### Вкажіть явно, що ви використовуєте DLE
Будь ласка, опублікуйте твіт зі згадкою [@Database_Lab](https://twitter.com/Database_Lab) або поділіться посиланням на цей репозиторій у вашій улюбленій соціальній мережі.

Якщо ви використовуєте DLE у роботі, подумайте, де ви могли б про це згадати. Один із найкращих способів згадування – використання графіки з посиланням. Деякі матеріали можна знайти у директорії `./assets`. Будь ласка, використовуйте їх у своїх документах, презентаціях, інтерфейсах програм та веб-сайтів, щоб показати, що ви використовуєте DLE.

HTML-код для світлих фонів:
<p>
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-light-background.svg" />
</p>

```html
<a href="http://databaselab.io">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-light-background.svg" />
</a>
```

Для темних фонів:
<p style="background-color: #bbb">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-dark-background.svg" />
</p>

```html
<a href="http://databaselab.io">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-dark-background.svg" />
</a>
```

### Запропонуйте ідею або повідомте про помилку
Детальніше: [CONTRIBUTING.md](../CONTRIBUTING.md).

### Беріть участь у розробці
Детальніше: [CONTRIBUTING.md](../CONTRIBUTING.md).

### Довідники
- [Компоненти DLE](https://postgres.ai/docs/reference-guides/database-lab-engine-components)
- [Довідник конфігурації DLE](https://postgres.ai/docs/database-lab/config-reference)
- [Довідник з DLE API](https://postgres.ai/swagger-ui/dblab/)
- [Довідник з Client CLI](https://postgres.ai/docs/database-lab/cli-reference)

### HowTo-інструкції
- [Як встановити Database Lab з Terraform на AWS](https://postgres.ai/docs/how-to-guides/administration/install-database-lab-with-terraform)
- [Як встановити та ініціалізувати Database Lab CLI](https://postgres.ai/docs/guides/cli/cli-install-init)
- [Як керувати DLE](https://postgres.ai/docs/how-to-guides/administration)
- [Як працювати з клонами](https://postgres.ai/docs/how-to-guides/cloning)

Ви можете знайти більше [секції "How-to guides"](https://postgres.ai/docs/how-to-guides) документації.

### Різне
- [Docker-образи DLE](https://hub.docker.com/r/postgresai/dblab-server)
- [Extended Docker images for PostgreSQL (з величезною кількістю розширень)](https://hub.docker.com/r/postgresai/extended-postgres)
- [Чатбот для оптимізації SQL (чатбот Joe)] (https://postgres.ai/docs/joe-bot)
- [DB Migration Checker](https://postgres.ai/docs/db-migration-checker)

## Ліцензія
Код DLE розповсюджується під ліцензією, схваленою OSI: [Apache 2.0](https://opensource.org/license/apache-2-0/).

Зв'яжіться з командою Postgres.ai, якщо вам потрібна комерційна ліцензія, яка не містить пунктів GPL, а також якщо вам потрібна підтримка: [Контактна сторінка](https://postgres.ai/contact).

## Спільнота та підтримка
- ["Кодекс поведінки спільноти Database Lab Engine"](../CODE_OF_CONDUCT.md)
- Де отримати допомогу: [Контактна сторінка](https://postgres.ai/contact)
- [Спільнота у Телеграм (російська мова)](https://t.me/databaselabru)
- [Спільнота у Slack](https://slack.postgres.ai)
- Якщо вам потрібно повідомити про проблему безпеки, дотримуйтесь інструкцій у документі ["SECURITY.md"](../SECURITY.md).

[![Кодекс поведінки](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?color=blue)](../CODE_OF_CONDUCT.md)

<!--
## Переводы
- ...
-->
