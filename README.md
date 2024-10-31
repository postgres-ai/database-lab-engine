<div align="center">
  <img width="500" src="./assets/dle.svg" border="0" />
  <sub><br /><a href="./translations/README.german.md">Deutsch</a> | <a href="./translations/README.portuguese-br.md">Português (BR)</a> | <a href="./translations/README.russian.md">Русский</a> | <a href="./translations/README.spanish.md">Español</a> | <a href="./translations/README.ukrainian.md">Українська</a></sub>
</div>

<br />

<div align="center"><h1 align="center">DBLab Engine</h1></div>

<div align="center">
  <a href="https://twitter.com/intent/tweet?via=Database_Lab&url=https://github.com/postgres-ai/database-lab-engine/&text=20@PostgreSQL%branching%20–%20DLE%20provides%20blazing-fast%20database%20cloning%20to%20build%20powerful%20development,%20test,%20QA,%20staging%20environments.">
    <img src="https://img.shields.io/twitter/url/https/github.com/postgres-ai/database-lab-engine.svg?style=for-the-badge" alt="twitter">
  </a>
</div>

<div align="center">
  <strong>⚡ Blazing-fast Postgres cloning and branching 🐘</strong><br /><br />
  🛠️ Build powerful dev/test environments.<br />
  🔃 Cover 100% of DB migrations with CI tests.<br>
  💡 Quickly verify ChatGPT ideas to get rid of hallucinations.<br /><br />
  Available for any PostgreSQL, including self-managed and managed<sup>*</sup> like AWS RDS, GCP CloudSQL, Supabase, Timescale.<br /><br />
  Can be installed and used anywhere: all clouds and on-premises.
</div>

<br />

<div align="center">
  <a href="https://postgres.ai" target="blank"><img src="https://img.shields.io/badge/Postgres-AI-orange.svg?style=flat" /></a> <a href="https://github.com/postgres-ai/database-lab-engine/releases/latest"><img src="https://img.shields.io/github/v/release/postgres-ai/database-lab-engine?color=orange&label=Database+Lab&logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAYCAYAAACWTY9zAAAACXBIWXMAAAsTAAALEwEAmpwYAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAPYSURBVHgBrVc9SCNBFH7JpVCrjdpotVgFES9qp8LdgaXNFWLnJY2lsVC0zIGKQeEujRw2508lNndqISKaA38a/4Io/qBGQc2B6IKgImLufYPj7W42Jsb9YNidb2ffvHnzzZsZB1mgra3to9Pp9Docjvdc9XJR3G63qm9zdXUV44fGJZZIJKKPj4+R/v7+CNkEh3wJBoPKzc1NIC8vr7WoqEgpLS2l4uJiYodEscLd3R2dnZ2Jcnh4SNvb23ByiG2E2R6cpo6Oju/s9EZfX9+Q/C8F95O5P5ITjnV2dqq5ubnz1dXVam1tLeXk5FA24CjS6uoqLS4uxtjpT729vbGLi4ujubk5lflf3IcfDuu4CHOfJbe8vKwuLCwITno7f3p6mrALBwcHCdiEba4egYP97u7uYDru8vIy0dPT8835NFg1Pz+f7MLT1Kt6DrIoKyv7ko7Dvx6Pxycdo3A4LKbirYDWRkdHLb/t7u5mxO3t7SkuWWlubhYGoa+qqiriBSBGlAkwoK2tLYhf1Ovr62lwcNDwfXJykgoLCzPiELVnx1BpaWkRK2xtbU2IGA3Bw1kWpMGZ29tb0jRNPNGmpKSE6urqxFOPgYEBcrlcwtmVlZWMOF48/x2TQJT0kZIpwQzpbKpUIuHz+YjTh4FrbGykgoKCFzmX3gGrNAHOHIXXwOwUYHbKinsWP+YWzr0VsDE+Pp7EQxZmoafisIAMGoNgkfFl1n8NMN0QP7RZU1Nj+IaOZmdnDUJ/iTOIH8LFasTHqakp0ZHUG6bTrCUpfk6I4h+0w4ACgYBoDxsAbzFUUVFBTU1NNDMzkxGH2TOIH53DORQZBdm5Ocehc6SUyspKQnJOtY21t7dnxSWtSj3MK/StQJQz4aDTZ/Fjbu2ClS1EfGdnJ4k7OTlJ4jBTLj2B1YRpzDY9SPHqp5WPUrS0tCQ64z3QwKG9FL+eM4i/oaFBkHzsoJGREeFcOvGfn5+LJ/7DO9rI7M9HKdFubGyMysvLBT8xMWHgsA1acQiQQWMwKKOFzuQBEOI35zg4gcyvKArhDCcHYIbf78+KSyl+vZN24f7+XjNzVuJHOyn+GCJjF5721pieQ+Ll8lvPoc/19fUkbnNzc1hEjC8dfj7yzHPGViH+dBtzKmC6oVEcrWETHJ+tKBqNwqlwKBQKWnCtVtw7kGxM83q9w8fHx3/ZqIdHrFxfX9PDw4PQEY4jVsBKhuhxFpuenkbR9vf3Q9ze39XVFUcb3sTd8Xj8K3f2Q/6XCeew6pBX1Ee+seD69oGrChfV6vrGR3SN22zg+sbXvQ2+fETIJvwDtXvnpBGzG2wAAAAASUVORK5CYII=" alt="Latest release" /></a>

  <a href="https://gitlab.com/postgres-ai/database-lab/-/pipelines" target="blank"><img src="https://gitlab.com/postgres-ai/database-lab//badges/master/pipeline.svg" alt="CI pipeline status" /></a> <a href="https://goreportcard.com/report/gitlab.com/postgres-ai/database-lab" target="blank"><img src="https://goreportcard.com/badge/gitlab.com/postgres-ai/database-lab" alt="Go report" /></a>  <a href="https://depshield.github.io" target="blank"><img src="https://depshield.sonatype.org/badges/postgres-ai/database-lab-engine/depshield.svg" alt="DepShield Badge" /></a>

  <a href="./CODE_OF_CONDUCT.md"><img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?logoColor=black&labelColor=white&color=blue" alt="Contributor Covenant" /></a> <a href="https://slack.postgres.ai" target="blank"><img src="https://img.shields.io/badge/Chat-Slack-blue.svg?logo=slack&style=flat&logoColor=black&labelColor=white&color=blue" alt="Community Slack" /></a> <a href="https://twitter.com/intent/follow?screen_name=Database_Lab" target="blank"><img src="https://img.shields.io/twitter/follow/Database_Lab.svg?style=social&maxAge=3600" alt="Twitter Follow" /></a>
</div>

<div align="center">
  <h3>
    <a href="#features">Features</a>
    <span> | </span>
    <a href="https://postgres.ai/docs">Documentation</a>
    <span> | </span>
    <a href="https://postgres.ai/blog/tags/database-lab-engine">Blog</a>
    <span> | </span>
    <a href="#community--support">Community & Support</a>
    <span> | </span>
    <a href="./CONTRIBUTING.md">Contributing</a>
  </h3>
</div>

---
  <sub><sup>*</sup>For managed PostgreSQL cloud services like AWS RDS or Heroku, direct physical connection and PGDATA access aren't possible. In these cases, DBLab should run on a separate VM within the same region. It will routinely auto-refresh its data, effectively acting as a database-as-a-service solution. This setup then offers thin database branching ideal for development and testing.</sub>

## Why DBLab?
- Build dev/QA/staging environments using full-scale, production-like databases.
- Provide temporary full-size database clones for SQL query analysis and optimization (see also: [SQL optimization chatbot Joe](https://gitlab.com/postgres-ai/joe)).
- Automatically test database changes in CI/CD pipelines, minimizing risks of production incidents.
- Rapidly validate ChatGPT or other LLM concepts, check for hallucinations, and iterate towards effective solutions.

For example, cloning a 1 TiB PostgreSQL database takes just about 10 seconds. On a single machine, you can have dozens of independent clones running simultaneously, supporting extensive development and testing activities without any added hardware costs.

<p><img src="./assets/dle-demo-animated.gif" border="0" /></p>

Try it yourself right now:
- Visit [Postgres.ai Console](https://console.postgres.ai/), set up your first organization and provision a DBLab Standard Edition (DBLab SE) to any cloud or on-prem
    - [Pricing](https://postgres.ai/pricing) (starting at $62/month)
    - [Doc: How to install DBLab SE](https://postgres.ai/docs/how-to-guides/administration/install-dle-from-postgres-ai)
- Demo: https://demo.aws.postgres.ai (use the token `demo-token` to access)
- Looking for a free version? Install DBLab Community Edition by [following this tutorial](https://postgres.ai/docs/tutorials/database-lab-tutorial)

## How it works
Thin cloning is fast because it is based on [Copy-on-Write (CoW)](https://en.wikipedia.org/wiki/Copy-on-write#In_computer_storage). DBLab employs two technologies for enabling thin cloning: [ZFS](https://en.wikipedia.org/wiki/ZFS) (default) and [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_(Linux)).

Using ZFS, DBLab routinely takes new snapshots of the data directory, managing a collection of them and removing old or unused ones. When requesting a fresh clone, users have the option to select their preferred snapshot.

Read more:
- [How it works](https://postgres.ai/products/how-it-works)
- [Database Migration Testing](https://postgres.ai/products/database-migration-testing)
- [SQL Optimization with Joe Bot](https://postgres.ai/products/joe)
- [Questions and answers](https://postgres.ai/docs/questions-and-answers)

## Where to start
- [DBLab tutorial for any PostgreSQL database](https://postgres.ai/docs/tutorials/database-lab-tutorial)
- [DBLab tutorial for Amazon RDS](https://postgres.ai/docs/tutorials/database-lab-tutorial-amazon-rds)
- [How to install DBLab SE using Postgres.ai Console](https://postgres.ai/docs/how-to-guides/administration/install-dle-from-postgres-ai)
- [How to install DBLab SE using AWS Marketplace](https://postgres.ai/docs/how-to-guides/administration/install-dle-from-aws-marketplace)

## Case studies
- GitLab: [How GitLab iterates on SQL performance optimization workflow to reduce downtime risks](https://postgres.ai/resources/case-studies/gitlab)

## Features
- Speed & scale
    - Blazing-fast cloning of Postgres databases – clone in seconds, irrespective of database size
    - Theoretical max of snapshots/clones: 2<sup>64</sup> ([ZFS](https://en.wikipedia.org/wiki/ZFS), default)
    - Maximum size of PostgreSQL data directory: 256 quadrillion zebibytes, or 2<sup>128</sup> bytes ([ZFS](https://en.wikipedia.org/wiki/ZFS), default)
- Support & technologies
    - Supported PostgreSQL versions: 9.6–17
    - Thin cloning ([CoW](https://en.wikipedia.org/wiki/Copy-on-write)) technologies: [ZFS](https://en.wikipedia.org/wiki/ZFS) and [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_(Linux))
    - UI for manual tasks and API & CLI for automation
    - Packaged in Docker containers for all components
- Postgres containers
    - Popular extensions including contrib modules, pgvector, HypoPG and many others ([docs](https://postgres.ai/docs/database-lab/supported-databases#extensions-included-by-default))
    - Customization capabilities for containers ([docs](https://postgres.ai/docs/database-lab/supported-databases#how-to-add-more-extensions))
    - Docker container and Postgres config parameters in DBLab config
- Source database requirements
    - Location flexibility: self-managed Postgres, AWS RDS, GCP CloudSQL, Azure, etc. No source adjustments needed
    - No ZFS or Docker requirements for source databases
- Data provisioning & retrieval
    - Physical (pg_basebackup, WAL-G, pgBackRest) and logical (dump/restore) provisioning
    - Partial data retrieval in logical mode (specific databases/tables)
    - Continuous update in physical mode
    - Periodic full refresh in logical mode without downtime
- Recovery & management
    - Fast Point in Time Recovery (PITR) for physical mode
    - Auto-deletion of unused clones
    - Snapshot retention policies in DBLab configuration
- Clones
    - "Deletion protection" for preventing clone deletion
    - Persistent clones withstand DBLab restarts
    - "Reset" command for data version switching
    - Resource quotas: CPU, RAM
- Monitoring & security
    - `/healthz` API endpoint (no auth), extended `/status` endpoint ([API docs](https://api.dblab.dev))
    - Netdata module for insights

## How to contribute
### Support us on GitHub/GitLab
The simplest way to show your support is by giving us a star on GitHub or GitLab! ⭐

![Add a star](./assets/star.gif)

### Spread the word
- Shoot out a tweet and mention [@Database_Lab](https://twitter.com/Database_Lab) 
- Share this repo's link on your favorite social media platform

### Share your experience
If DBLab has been a vital tool for you, tell the world about your journey. Use the logo from the `./assets` folder for a visual touch. Whether it's in documents, presentations, applications, or on your website, let everyone know you trust and use DBLab.

HTML snippet for lighter backgrounds:
<p>
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-light-background.svg" />
</p>

```html
<a href="http://databaselab.io">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-light-background.svg" />
</a>
```

For darker backgrounds:
<p style="background-color: #bbb">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-dark-background.svg" />
</p>

```html
<a href="http://databaselab.io">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-dark-background.svg" />
</a>
```

### Propose an idea or report a bug
Check out our [contributing guide](./CONTRIBUTING.md) for more details.

### Participate in development
Check out our [contributing guide](./CONTRIBUTING.md) for more details.


### Reference guides
- [DBLab components](https://postgres.ai/docs/reference-guides/database-lab-engine-components)
- [Client CLI reference](https://postgres.ai/docs/database-lab/cli-reference)
- [DBLab API reference](https://api.dblab.dev/)
- [DBLab configuration reference](https://postgres.ai/docs/database-lab/config-reference)

### How-to guides
- [How to install and initialize Database Lab CLI](https://postgres.ai/docs/how-to-guides/cli/cli-install-init)
- [How to manage DBLab](https://postgres.ai/docs/how-to-guides/administration)
- [How to work with clones](https://postgres.ai/docs/how-to-guides/cloning)

More you can find in [the "How-to guides" section](https://postgres.ai/docs/how-to-guides) of the docs. 

### Miscellaneous
- [DBLab Docker images](https://hub.docker.com/r/postgresai/dblab-server)
- [Extended Docker images for PostgreSQL (with plenty of extensions)](https://hub.docker.com/r/postgresai/extended-postgres)
- [SQL Optimization chatbot (Joe Bot)](https://postgres.ai/docs/joe-bot)
- [DB Migration Checker](https://postgres.ai/docs/db-migration-checker)

## License
DBLab source code is licensed under the OSI-approved open source license [Apache 2.0](https://opensource.org/license/apache-2-0/).

Reach out to the Postgres.ai team if you want a trial or commercial license that does not contain the GPL clauses: [Contact page](https://postgres.ai/contact).

## Community & Support
- ["Database Lab Engine Community Covenant Code of Conduct"](./CODE_OF_CONDUCT.md)
- Where to get help: [Contact page](https://postgres.ai/contact)
- [Community Slack](https://slack.postgres.ai)
- If you need to report a security issue, follow instructions in ["Database Lab Engine security guidelines"](./SECURITY.md)

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?color=blue)](./CODE_OF_CONDUCT.md)

Many thanks to our amazing contributors!

<a href = "https://github.com/postgres-ai/database-lab/graphs/contributors">
  <img src = "https://contrib.rocks/image?repo=postgres-ai/database-lab"/>
</a>

## Translations
Making DBLab more accessible to engineers around the globe is a great help for the project. Check details in the [translation section of contributing guide](./CONTRIBUTING.md#Translation).

This README is available in the following translations:
- [German / Deutsch](translations/README.german.md) (by [@ane4ka](https://github.com/ane4ka))
- [Brazilian Portuguese / Português (BR)](translations/README.portuguese-br.md) (by [@Alexand](https://gitlab.com/Alexand))
- [Russian / Pусский](translations/README.russian.md) (by [@Tanya301](https://github.com/Tanya301))
- [Spanish / Español](translations/README.spanish.md) (by [@asotolongo](https://gitlab.com/asotolongo))
- [Ukrainian / Українська](translations/README.ukrainian.md) (by [@denis-boost](https://github.com/denis-boost))

👉 [How to make a translation contribution](./CONTRIBUTING.md#translation)


