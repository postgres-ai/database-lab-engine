<div align="center">
  <img width="500" src="../assets/dle.svg" border="0" />
  <sub><br /><a href="./README.german.md">Deutsch</a> | <a href="./README.portuguese-br.md">Português (BR)</a> | <a href="./README.russian.md">Русский</a> | <a href="./README.spanish.md">Español</a> | <a href="./README.ukrainian.md">Українська</a></sub>
</div>

<br />

<div align="center"><h1 align="center">Database Lab Engine (DLE)</h1></div>

<div align="center">
  <a href="https://twitter.com/intent/tweet?via=Database_Lab&url=https://github.com/postgres-ai/database-lab-engine/&text=Thin%20@PostgreSQL%20clones%20–%20DLE%20provides%20blazing-fast%20database%20cloning%20to%20build%20powerful%20development,%20test,%20QA,%20staging%20environments.">
    <img src="https://img.shields.io/twitter/url/https/github.com/postgres-ai/database-lab-engine.svg?style=for-the-badge" alt="twitter">
  </a>
</div>

<div align="center">
  <strong>:zap: Clonagem ultrarrápida de bancos de dados PostgreSQL :elephant:</strong><br>
  Thin clones de bancos de dados PostgreSQL para viabilizar desenvolvimento, testes, QA e ambientes de staging poderosos.<br>
  <sub>Disponível para qualquer PostgreSQL, incluindo AWS RDS, GCP CloudSQL, Heroku, Digital Ocean e instâncias autoadministradas.</sub>
</div>

<br />

<div align="center">
  <a href="https://postgres.ai" target="blank"><img src="https://img.shields.io/badge/Postgres-AI-orange.svg?style=flat" /></a> <a href="https://github.com/postgres-ai/database-lab-engine/releases/latest"><img src="https://img.shields.io/github/v/release/postgres-ai/database-lab-engine?color=orange&label=Database+Lab&logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAYCAYAAACWTY9zAAAACXBIWXMAAAsTAAALEwEAmpwYAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAPYSURBVHgBrVc9SCNBFH7JpVCrjdpotVgFES9qp8LdgaXNFWLnJY2lsVC0zIGKQeEujRw2508lNndqISKaA38a/4Io/qBGQc2B6IKgImLufYPj7W42Jsb9YNidb2ffvHnzzZsZB1mgra3to9Pp9Docjvdc9XJR3G63qm9zdXUV44fGJZZIJKKPj4+R/v7+CNkEh3wJBoPKzc1NIC8vr7WoqEgpLS2l4uJiYodEscLd3R2dnZ2Jcnh4SNvb23ByiG2E2R6cpo6Oju/s9EZfX9+Q/C8F95O5P5ITjnV2dqq5ubnz1dXVam1tLeXk5FA24CjS6uoqLS4uxtjpT729vbGLi4ujubk5lflf3IcfDuu4CHOfJbe8vKwuLCwITno7f3p6mrALBwcHCdiEba4egYP97u7uYDru8vIy0dPT8835NFg1Pz+f7MLT1Kt6DrIoKyv7ko7Dvx6Pxycdo3A4LKbirYDWRkdHLb/t7u5mxO3t7SkuWWlubhYGoa+qqiriBSBGlAkwoK2tLYhf1Ovr62lwcNDwfXJykgoLCzPiELVnx1BpaWkRK2xtbU2IGA3Bw1kWpMGZ29tb0jRNPNGmpKSE6urqxFOPgYEBcrlcwtmVlZWMOF48/x2TQJT0kZIpwQzpbKpUIuHz+YjTh4FrbGykgoKCFzmX3gGrNAHOHIXXwOwUYHbKinsWP+YWzr0VsDE+Pp7EQxZmoafisIAMGoNgkfFl1n8NMN0QP7RZU1Nj+IaOZmdnDUJ/iTOIH8LFasTHqakp0ZHUG6bTrCUpfk6I4h+0w4ACgYBoDxsAbzFUUVFBTU1NNDMzkxGH2TOIH53DORQZBdm5Ocehc6SUyspKQnJOtY21t7dnxSWtSj3MK/StQJQz4aDTZ/Fjbu2ClS1EfGdnJ4k7OTlJ4jBTLj2B1YRpzDY9SPHqp5WPUrS0tCQ64z3QwKG9FL+eM4i/oaFBkHzsoJGREeFcOvGfn5+LJ/7DO9rI7M9HKdFubGyMysvLBT8xMWHgsA1acQiQQWMwKKOFzuQBEOI35zg4gcyvKArhDCcHYIbf78+KSyl+vZN24f7+XjNzVuJHOyn+GCJjF5721pieQ+Ll8lvPoc/19fUkbnNzc1hEjC8dfj7yzHPGViH+dBtzKmC6oVEcrWETHJ+tKBqNwqlwKBQKWnCtVtw7kGxM83q9w8fHx3/ZqIdHrFxfX9PDw4PQEY4jVsBKhuhxFpuenkbR9vf3Q9ze39XVFUcb3sTd8Xj8K3f2Q/6XCeew6pBX1Ee+seD69oGrChfV6vrGR3SN22zg+sbXvQ2+fETIJvwDtXvnpBGzG2wAAAAASUVORK5CYII=" alt="Latest release" /></a>

  <a href="https://gitlab.com/postgres-ai/database-lab/-/pipelines" target="blank"><img src="https://gitlab.com/postgres-ai/database-lab//badges/master/pipeline.svg" alt="CI pipeline status" /></a> <a href="https://goreportcard.com/report/gitlab.com/postgres-ai/database-lab" target="blank"><img src="https://goreportcard.com/badge/gitlab.com/postgres-ai/database-lab" alt="Go report" /></a>  <a href="https://depshield.github.io" target="blank"><img src="https://depshield.sonatype.org/badges/postgres-ai/database-lab-engine/depshield.svg" alt="DepShield Badge" /></a>

  <a href="../CODE_OF_CONDUCT.md"><img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?logoColor=black&labelColor=white&color=blue" alt="Contributor Covenant" /></a> <a href="https://slack.postgres.ai" target="blank"><img src="https://img.shields.io/badge/Chat-Slack-blue.svg?logo=slack&style=flat&logoColor=black&labelColor=white&color=blue" alt="Community Slack" /></a> <a href="https://twitter.com/intent/follow?screen_name=Database_Lab" target="blank"><img src="https://img.shields.io/twitter/follow/Database_Lab.svg?style=social&maxAge=3600" alt="Twitter Follow" /></a>
</div>

<div align="center">
  <h3>
    <a href="#features">Funcionalidades</a>
    <span> | </span>
    <a href="https://postgres.ai/docs">Documentação</a>
    <span> | </span>
    <a href="https://postgres.ai/blog/tags/database-lab-engine">Blog</a>
    <span> | </span>
    <a href="#community--support">Comunidade & Suporte</a>
    <span> | </span>
    <a href="../CONTRIBUTING.md">Contribuindo</a>
  </h3>
</div>

## Por que DLE?
- Construa ambientes de dev/QA/staging baseados nos bancos de dados de produção completos.
- Disponibilize clones temporários completos dos bancos de dados de produção para análises de queries e otimizações (veja também: [SQL optimization chatbot Joe](https://gitlab.com/postgres-ai/joe)).
- Faça testes automatizados em pipelines de integração contínua para evitar incidentes em produção.

Por examplo, clonar um banco de dados PostgreSQL de 1 TiB dura ~10 segundos. Dezenas de clones independentes são iniciados numa mesma máquina, suportando vários atividades de desenvolvimento e teste, sem aumento de custo de hardware.

<p><img src="../assets/dle-demo-animated.gif" border="0" /></p>

Teste agora mesmo:
- entre no [Database Lab Platform](https://console.postgres.ai/), associe-se a uma organização "Demo", e teste clonar um banco de dados demo de ~1TiB, ou
- faça o check out de um outro setup demo, DLE CE: https://nik-tf-test.aws.postgres.ai:446/instance, use o token `demo` para acessar (este setup tem certificados autoassinado, então ignore os alertas do navegador)

## Como funciona
Thin cloning é rápido pois utiliza [Copy-on-Write (CoW)](https://en.wikipedia.org/wiki/Copy-on-write#In_computer_storage). DLE suporta duas tecnologias para abilitar CoW e thin cloning: [ZFS](https://en.wikipedia.org/wiki/ZFS) (default) e [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_(Linux)).

Com ZFS, o Database Lab Engine periodicamente cria um novo snapshot do diretório de dados e mantém um conjunto de snapshots, limpando os antigos e não utilizados. Quando solicitam um novo clone, usuários podem escolher qual snapshot utilizar.

Leia mais:
- [Como funciona](https://postgres.ai/products/how-it-works)
- [Testando Database Migrations](https://postgres.ai/products/database-migration-testing)
- [Otimização SQL com Joe Bot](https://postgres.ai/products/joe)
- [Perguntas e Respostas](https://postgres.ai/docs/questions-and-answers)

## Onde começar
- [Tutorial do Database Lab para qualquer banco de dados PostgreSQL](https://postgres.ai/docs/tutorials/database-lab-tutorial)
- [Tutorial do Database Lab para Amazon RDS](https://postgres.ai/docs/tutorials/database-lab-tutorial-amazon-rds)
- [Template com Terraform module (AWS)](https://postgres.ai/docs/how-to-guides/administration/install-database-lab-with-terraform)

## Estudos de Caso
- Qiwi: [Como o Qiwi controla os dados para acelerar o desenvolvimento](https://postgres.ai/resources/case-studies/qiwi)
- GitLab: [Como o GitLab itera na otimização de performances SQL para reduzir os riscos de downtime](https://postgres.ai/resources/case-studies/gitlab)

## Funcionalidades
- Clonagem the bancos de dados Postgres ultrarrápidos - apenas alguns segundos para criar um novo clone pronto para aceitar conexões e queries, independentemente do tamanho do banco de dados.
- O número máximo teórico de snapshots e clones é 2<sup>64</sup> ([ZFS](https://en.wikipedia.org/wiki/ZFS), default).
- O número máximo teórico de do diretório de dados do PostgreSQL: 256 quatrilhões zebibytes, ou 2<sup>128</sup> bytes ([ZFS](https://en.wikipedia.org/wiki/ZFS), default).
- Versões _major_ do PostgreSQL suportadas: 9.6–14.
- Duas tecnologias são suportadas para viabilizar o thin cloning ([CoW](https://en.wikipedia.org/wiki/Copy-on-write)): [ZFS](https://en.wikipedia.org/wiki/ZFS) e [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_(Linux)).
- Todos os componentes estão empacotados em docker containers.
- UI para tornar o trabalho manual mais conveniente.
- API e CLI para automatizar o trabalho com DLE snapshots e clones.
- Por default, os PostgreSQL containers incluem várias extensões populares ([docs](https://postgres.ai/docs/database-lab/supported-databases#extensions-included-by-default)).
- PostgreSQL containers podem ser customizados ([docs](https://postgres.ai/docs/database-lab/supported-databases#how-to-add-more-extensions)).
- O banco de dados original pode estar localizado em qualquer lugar (Postgres autoadministrado, AWS RDS, GCP CloudSQL, Azure, Timescale Cloud, etc) e NÃO requer nenhum ajuste. Não há NENHUM requerimento para instalar o ZFS ou Docker nos bancos de dados originais (production).
- Um provisionamento de dados inicial pode ser feito tanto no nível físico (pg_basebackup, backup / ferramentes de arquivamento como WAL-G ou pgBackRest) ou lógico (dump/restore direto da origem ou de arquivos armazenados na AWS S3).
- Para o modo lógico, suporta a retenção parcial de dados (bancos específicos, tabelas específicas).
- Para o modo físico, um estado de atualização contínua é suportado ("sync container"), tornando o DLE uma versão especializada de um standby Postgres.
- Para o modo lógico, suporta atualização completa periódica, automatizada, e controlada pelo DLE. É possível utilizar multiplos discos contendo diferentes versões do banco de dados, para que a atualização completa não precise de _downtime_.
- Fast Point in Time Recovery (PITR) para os pontos disponíveis em DLE _snapshots_.
- Clones não utilizados são automaticamente removidos.
- "Deletion protection" _flag_ pode ser utilizada para bloquear remoções automáticas ou manuais dos clones.
- _Snapshot retention policies_ são suportadas na configuração do DLE.
- Clones Persistentes: clones sobrevivem a DLE _restarts_ (incluindo _reboot_ total da VM).
- O comand "reset" pode ser utilizado para trocar para uma versão diferente dos dados.
- O componente DB Migration Checker coleta vários artefatos úteis para testar o banco de dados em CI ([docs](https://postgres.ai/docs/db-migration-checker)).
- SSH port forwarding para conexões com a API e o Postgres.
- É possível especificar parâmetros para configurar Docker _containers_ na configuração do DLE.
- Cotas de utilização de recursos para clones: CPU, RAM (cotas de container, suportadas pelo Docker)
- Parâmetros de configuração do Postgres podem ser especificados na configuração do DLE (separadamente para clones, o "sync" _container_, e o "promote" _container_).
- Monitoramento: `/healthz` API _endpoint_ livre de autenticação, `/status` extendido (requer autenticação), [Netdata module](https://gitlab.com/postgres-ai/netdata_for_dle).

## Contribuindo
### Adicione um estrela ao projeto
A forma mais simples the contribuir é adicionar uma estrela ao projeto no GitHub/GitLab:

![Adicionar estrela](../assets/star.gif)

### Compartilhe o projeto
Poste um tweet mencionando [@Database_Lab](https://twitter.com/Database_Lab) ou compartilhe o linke para este repositório na sua rede social favorita.

Se você usa o DLE ativamente, conte aos outros sobre a sua experiência. Você pode usar o logo que está referenciado abaixo salvo na pasta `./assets`. Fique à vontade para por nos seus documentos, apresentações, aplicações, e interfaces web para mostrar que você utiliza o DLE.

_Snippet_ HTML para _backgrounds_ mais claros:
<p>
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-light-background.svg" />
</p>

```html
<a href="http://databaselab.io">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-light-background.svg" />
</a>
```

Para _backgrounds_ mais escuros:
<p style="background-color: #bbb">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-dark-background.svg" />
</p>

```html
<a href="http://databaselab.io">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-dark-background.svg" />
</a>
```

### Proponha uma idéia ou reporte um _bug_
Veja o nosso [guia de contribuição](../CONTRIBUTING.md) para mais detalhes.

### Participate in development
Veja o nosso [guia de contribuição](../CONTRIBUTING.md) para mais detalhes.

### Traduza o README
Tornar o Database Lab Engine mais acessível aos engenheiros no mundo todo é uma ótima ajuda para o projeto. Veja detalhes na [seção de tradução do guia de contribuição](../CONTRIBUTING.md#Translation).

### Referências
- [Componentes DLE](https://postgres.ai/docs/reference-guides/database-lab-engine-components)
- [Configuração do DLE](https://postgres.ai/docs/database-lab/config-reference)
- [Documentação da API](https://postgres.ai/swagger-ui/dblab/)
- [Documentação do Client CLI](https://postgres.ai/docs/database-lab/cli-reference)

### How-to guides
- [Como instalar o Database Lab com Terraform na AWS](https://postgres.ai/docs/how-to-guides/administration/install-database-lab-with-terraform)
- [Como instalar e inicializar o Database Lab CLI](https://postgres.ai/docs/guides/cli/cli-install-init)
- [Como administrar o DLE](https://postgres.ai/docs/how-to-guides/administration)
- [Como trabalhar com clones](https://postgres.ai/docs/how-to-guides/cloning)

Você pode encontrar mais [no seção "How-to guides"](https://postgres.ai/docs/how-to-guides) dos documentos.

### Diversos
- [Imagens Docker do DLE](https://hub.docker.com/r/postgresai/dblab-server)
- [Imagens Docker extendidas para PostgreSQL (com uma penca de extensões)](https://hub.docker.com/r/postgresai/extended-postgres)
- [SQL Optimization chatbot (Joe Bot)](https://postgres.ai/docs/joe-bot)
- [DB Migration Checker](https://postgres.ai/docs/db-migration-checker)

## Licença
O código fonte do DLE está licensiado pela licença de código aberto GNU Affero General Public License version 3 (AGPLv3), aprovada pela OSI.

Contacte o time do Postgres.ai se você desejar uma licença _trial_ ou comercial que não contenha as cláusulas da GPL: [Página de contato](https://postgres.ai/contact).

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fpostgres-ai%2Fdatabase-lab-engine.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fpostgres-ai%2Fdatabase-lab-engine?ref=badge_large)

## Comunidade e Suporte
- ["Acordo da Comunidade do Database Lab Engine de Código de Conduta"](../CODE_OF_CONDUCT.md)
- Onde conseguir ajuda: [Página de contato](https://postgres.ai/contact)
- [Comunidade no Slack](https://slack.postgres.ai)
- Se você precisa reportar um problema de segurança, siga as instruções em ["Database Lab Engine guia de segurança"](../SECURITY.md).

[![Acordo do contribuidor](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?color=blue)](../CODE_OF_CONDUCT.md)
