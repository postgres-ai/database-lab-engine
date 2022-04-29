<div align="center">
  <img width="500" src="../assets/dle.svg" border="0" />
  <sub><br /><a href="./README.german.md">Deutsch</a> | <a href="./README.portuguese-br.md">Português (BR)</a> | <a href="./README.russian.md">Русский</a> | <a href="./README.spanish.md">Español</a> | <a href="./README.ukrainian.md">Українська</a></sub>
</div>

<br />

<div align="center"><img width="500" src="../assets/dle.svg" border="0" /></div>

<div align="center"><h1 align="center">Database Lab Engine (DLE)</h1></div>

<div align="center">
  <a href="https://twitter.com/intent/tweet?via=Database_Lab&url=https://github.com/postgres-ai/database-lab-engine/&text=Thin%20@PostgreSQL%20clones%20–%20DLE%20provides%20blazing-fast%20database%20cloning%20to%20build%20powerful%20development,%20test,%20QA,%20staging%20environments.">
    <img src="https://img.shields.io/twitter/url/https/github.com/postgres-ai/database-lab-engine.svg?style=for-the-badge" alt="twitter">
  </a>
</div>

<div align="center">
  <strong>:zap: Clonación ultrarrápida de bases de datos PostgreSQL :elephant:</strong><br>
  Clones delgados de PostgreSQL para crear potentes entornos de desarrollo, prueba, control de calidad y ensayo.<br>
  <sub>Disponible para cualquier PostgreSQL, incluidos AWS RDS, GCP CloudSQL, Heroku, Digital Ocean e instancias autoadministradas.</sub>
</div>

<br />

<div align="center">
  <a href="https://postgres.ai" target="blank"><img src="https://img.shields.io/badge/Postgres-AI-orange.svg?style=flat" /></a> <a href="https://github.com/postgres-ai/database-lab-engine/releases/latest"><img src="https://img.shields.io/github/v/release/postgres-ai/database-lab-engine?color=orange&label=Database+Lab&logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAYCAYAAACWTY9zAAAACXBIWXMAAAsTAAALEwEAmpwYAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAPYSURBVHgBrVc9SCNBFH7JpVCrjdpotVgFES9qp8LdgaXNFWLnJY2lsVC0zIGKQeEujRw2508lNndqISKaA38a/4Io/qBGQc2B6IKgImLufYPj7W42Jsb9YNidb2ffvHnzzZsZB1mgra3to9Pp9Docjvdc9XJR3G63qm9zdXUV44fGJZZIJKKPj4+R/v7+CNkEh3wJBoPKzc1NIC8vr7WoqEgpLS2l4uJiYodEscLd3R2dnZ2Jcnh4SNvb23ByiG2E2R6cpo6Oju/s9EZfX9+Q/C8F95O5P5ITjnV2dqq5ubnz1dXVam1tLeXk5FA24CjS6uoqLS4uxtjpT729vbGLi4ujubk5lflf3IcfDuu4CHOfJbe8vKwuLCwITno7f3p6mrALBwcHCdiEba4egYP97u7uYDru8vIy0dPT8835NFg1Pz+f7MLT1Kt6DrIoKyv7ko7Dvx6Pxycdo3A4LKbirYDWRkdHLb/t7u5mxO3t7SkuWWlubhYGoa+qqiriBSBGlAkwoK2tLYhf1Ovr62lwcNDwfXJykgoLCzPiELVnx1BpaWkRK2xtbU2IGA3Bw1kWpMGZ29tb0jRNPNGmpKSE6urqxFOPgYEBcrlcwtmVlZWMOF48/x2TQJT0kZIpwQzpbKpUIuHz+YjTh4FrbGykgoKCFzmX3gGrNAHOHIXXwOwUYHbKinsWP+YWzr0VsDE+Pp7EQxZmoafisIAMGoNgkfFl1n8NMN0QP7RZU1Nj+IaOZmdnDUJ/iTOIH8LFasTHqakp0ZHUG6bTrCUpfk6I4h+0w4ACgYBoDxsAbzFUUVFBTU1NNDMzkxGH2TOIH53DORQZBdm5Ocehc6SUyspKQnJOtY21t7dnxSWtSj3MK/StQJQz4aDTZ/Fjbu2ClS1EfGdnJ4k7OTlJ4jBTLj2B1YRpzDY9SPHqp5WPUrS0tCQ64z3QwKG9FL+eM4i/oaFBkHzsoJGREeFcOvGfn5+LJ/7DO9rI7M9HKdFubGyMysvLBT8xMWHgsA1acQiQQWMwKKOFzuQBEOI35zg4gcyvKArhDCcHYIbf78+KSyl+vZN24f7+XjNzVuJHOyn+GCJjF5721pieQ+Ll8lvPoc/19fUkbnNzc1hEjC8dfj7yzHPGViH+dBtzKmC6oVEcrWETHJ+tKBqNwqlwKBQKWnCtVtw7kGxM83q9w8fHx3/ZqIdHrFxfX9PDw4PQEY4jVsBKhuhxFpuenkbR9vf3Q9ze39XVFUcb3sTd8Xj8K3f2Q/6XCeew6pBX1Ee+seD69oGrChfV6vrGR3SN22zg+sbXvQ2+fETIJvwDtXvnpBGzG2wAAAAASUVORK5CYII=" alt="Latest release" /></a>

  <a href="https://gitlab.com/postgres-ai/database-lab/-/pipelines" target="blank"><img src="https://gitlab.com/postgres-ai/database-lab//badges/master/pipeline.svg" alt="CI pipeline status" /></a> <a href="https://goreportcard.com/report/gitlab.com/postgres-ai/database-lab" target="blank"><img src="https://goreportcard.com/badge/gitlab.com/postgres-ai/database-lab" alt="Go report" /></a>  <a href="https://depshield.github.io" target="blank"><img src="https://depshield.sonatype.org/badges/postgres-ai/database-lab-engine/depshield.svg" alt="DepShield Badge" /></a>

  <a href="../CODE_OF_CONDUCT.md"><img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?logoColor=black&labelColor=white&color=blue" alt="Contributor Covenant" /></a> <a href="https://slack.postgres.ai" target="blank"><img src="https://img.shields.io/badge/Chat-Slack-blue.svg?logo=slack&style=flat&logoColor=black&labelColor=white&color=blue" alt="Community Slack" /></a> <a href="https://twitter.com/intent/follow?screen_name=Database_Lab" target="blank"><img src="https://img.shields.io/twitter/follow/Database_Lab.svg?style=social&maxAge=3600" alt="Twitter Follow" /></a>
</div>

<div align="center">
  <h3>
    <a href="#características">Características</a>
    <span> | </span>
    <a href="https://postgres.ai/docs">Documentación</a>
    <span> | </span>
    <a href="https://postgres.ai/blog/tags/database-lab-engine">Blog</a>
    <span> | </span>
    <a href="#comunidad--apoyo">Comunidad & Apoyo</a>
    <span> | </span>
    <a href="../CONTRIBUTING.md">Contribuyendo</a>
  </h3>
</div>

## ¿Por qué DLE?
- Cree entornos de desarrollo, control de calidad y puesta en escena basados en bases de datos de producción de tamaño completo.
- Proporcione clones temporales de bases de datos de tamaño completo para el análisis y la optimización de consultas SQL (ver también: [Joe, chatbot de optimización SQL](https://gitlab.com/postgres-ai/joe)).
- Pruebe automáticamente los cambios de la base de datos en las tuberías de CI/CD para evitar incidentes en producción.

Por ejemplo, la clonación de una base de datos PostgreSQL de 1 TiB tarda unos 10 segundos. Docenas de clones independientes están en funcionamiento en una sola máquina, lo que respalda muchas actividades de desarrollo y prueba, sin aumentar los costos de hardware.

<p><img src="../assets/dle-demo-animated.gif" border="0" /></p>

Pruébelo usted mismo ahora mismo:
- Ingrese a [la plataforma de laboratorio de base de datos](https://console.postgres.ai/), únase a la organización "Demo" y pruebe la clonación de la base de datos de demostración de ~1 TiB.
- Vea otra configuración de demostración, DLE CE: https://nik-tf-test.aws.postgres.ai:446/instance, use el token `demo` para ingresar (esta configuración tiene certificados autofirmados, así que ignore los certificados del navegador) quejas)

## Cómo funciona
La clonación ligera es rápida porque usa [Copy-on-Write (CoW)](https://en.wikipedia.org/wiki/Copy-on-write#In_computer_storage). DLE admite dos tecnologías para habilitar CoW y clonación ligera: [ZFS](https://en.wikipedia.org/wiki/ZFS) (predeterminado) y [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_ (Linux)).

Con ZFS, Database Lab Engine crea periódicamente una nueva instantánea del directorio de datos y mantiene un conjunto de instantáneas, limpiando las antiguas y no utilizadas. Al solicitar un nuevo clon, los usuarios pueden elegir qué instantánea usar.

Lee más:
- [Cómo funciona](https://postgres.ai/products/how-it-works)
- [Pruebas de migración de bases de datos](https://postgres.ai/products/database-migration-testing)
- [Optimización SQL con Joe Bot](https://postgres.ai/products/joe)
- [Preguntas y respuestas](https://postgres.ai/docs/questions-and-answers)

## Donde empezar
- [Tutorial de laboratorio de base de datos para cualquier base de datos PostgreSQL](https://postgres.ai/docs/tutorials/database-lab-tutorial)
- [Tutorial de laboratorio de base de datos para Amazon RDS](https://postgres.ai/docs/tutorials/database-lab-tutorial-amazon-rds)
- [Plantilla del módulo Terraform (AWS)](https://postgres.ai/docs/how-to-guides/administration/install-database-lab-with-terraform)

## Estudios de caso
- Qiwi: [Cómo controla Qiwi los datos para acelerar el desarrollo](https://postgres.ai/resources/case-studies/qiwi)
- GitLab: [Cómo itera GitLab en el flujo de trabajo de optimización del rendimiento de SQL para reducir los riesgos de tiempo de inactividad](https://postgres.ai/resources/case-studies/gitlab)

## Características
- Clonación ultrarrápida de bases de datos de Postgres: unos segundos para crear un nuevo clon listo para aceptar conexiones y consultas, independientemente del tamaño de la base de datos.
- El número máximo teórico de instantáneas y clones es 2<sup>64</sup> ([ZFS](https://en.wikipedia.org/wiki/ZFS), predeterminado).
- El tamaño máximo teórico del directorio de datos de PostgreSQL: 256 cuatrillones de zebibytes, o 2<sup>128</sup> bytes ([ZFS](https://en.wikipedia.org/wiki/ZFS), predeterminado).
- Versiones principales de PostgreSQL admitidas: 9.6–14.
- Se admiten dos tecnologías para permitir la clonación ligera ([CoW](https://en.wikipedia.org/wiki/Copy-on-write)): [ZFS](https://en.wikipedia.org/wiki/ZFS) y [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_(Linux)).
- Todos los componentes están empaquetados en contenedores Docker.
- Interfaz de usuario para que el trabajo manual sea más conveniente.
- API y CLI para automatizar el trabajo con instantáneas y clones de DLE.
- De forma predeterminada, los contenedores de PostgreSQL incluyen muchas extensiones populares ([docs](https://postgres.ai/docs/database-lab/supported-databases#extensions-included-by-default)).
- Los contenedores de PostgreSQL se pueden personalizar ([docs](https://postgres.ai/docs/database-lab/supported-databases#how-to-add-more-extensions)).
- La base de datos de origen se puede ubicar en cualquier lugar (Postgres autoadministrado, AWS RDS, GCP CloudSQL, Azure, Timescale Cloud, etc.) y NO requiere ningún ajuste. NO hay requisitos para instalar ZFS o Docker en las bases de datos de origen (producción).
- El aprovisionamiento de datos inicial puede ser físico (pg_basebackup, herramientas de copia de seguridad/archivo como WAL-G o pgBackRest) o lógico (volcado/restauración directamente desde el origen o desde archivos almacenados en AWS S3).
- Para el modo lógico, se admite la recuperación parcial de datos (bases de datos específicas, tablas específicas).
- Para el modo físico, se admite un estado actualizado continuamente ("contenedor de sincronización"), lo que convierte a DLE en una versión especializada de Postgres en espera.
- Para el modo lógico, la actualización completa periódica es compatible, automatizada y controlada por DLE. Es posible usar varios discos que contengan diferentes versiones de la base de datos, por lo que la actualización completa no requerirá tiempo de inactividad.
- Recuperación rápida de un punto en el tiempo (PITR) a los puntos disponibles en las instantáneas DLE.
- Los clones no utilizados se eliminan automáticamente.
- El indicador de "Protección de eliminación" se puede utilizar para bloquear la eliminación automática o manual de clones.
- Políticas de retención de instantáneas admitidas en la configuración de DLE.
- Clones persistentes: los clones sobreviven a los reinicios de DLE (incluidos los reinicios completos de VM).
- El comando "restablecer" se puede usar para cambiar a una versión diferente de los datos.
- El componente DB Migration Checker recopila varios artefactos útiles para las pruebas de base de datos en CI ([docs](https://postgres.ai/docs/db-migration-checker)).
- Reenvío de puertos SSH para conexiones API y Postgres.
- Los parámetros de configuración del contenedor de Docker se pueden especificar en la configuración de DLE.
- Cuotas de uso de recursos para clones: CPU, RAM (cuotas de contenedores, compatibles con Docker)
- Los parámetros de configuración de Postgres se pueden especificar en la configuración de DLE (por separado para los clones, el contenedor de "sincronización" y el contenedor de "promoción").
- Supervisión: extremo de API `/healthz` sin autenticación, `/status` extendido (requiere autenticación), [módulo Netdata] (https://gitlab.com/postgres-ai/netdata_for_dle).

## Como contribuir
### Ponle una estrella al proyecto
Si te gusta Database Lab Engine, ¡ayúdanos con una estrella en GitHub/GitLab!

![Darle una estrella de GitHub/GitLab](../assets/star.gif)

### Menciona que usas DLE
Publique un tweet mencionando [@Database_Lab](https://twitter.com/Database_Lab) o comparta el enlace a este repositorio en su red social favorita.

Si está utilizando activamente DLE en el trabajo, piense dónde podría mencionarlo. La mejor manera de mencionarlo es usando gráficos con un enlace. Los activos de la marca se pueden encontrar en la carpeta `./assets`. Siéntase libre de incluirlos en sus documentos, presentaciones de diapositivas, aplicaciones e interfaces de sitios web para demostrar que usa DLE.

Fragmento de HTML para fondos más claros:
<p>
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-light-background.svg" />
</p>

```html
<a href="http://databaselab.io">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-light-background.svg" />
</a>
```

Para fondos más oscuros:
<p style="background-color: #bbb">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-dark-background.svg" />
</p>

```html
<a href="http://databaselab.io">
  <img width="400" src="https://postgres.ai/assets/powered-by-dle-for-dark-background.svg" />
</a>
```

### Proponer una idea o reportar un error
Consulte nuestra [guía de contribución](../CONTRIBUTING.md) para obtener más detalles.

### Participar en el desarrollo
Consulte nuestra [guía de contribución](../CONTRIBUTING.md) para obtener más detalles.

### Guías de referencia
- [componentes DLE](https://postgres.ai/docs/reference-guides/database-lab-engine-components)
- [Referencia de configuración de DLE](https://postgres.ai/docs/database-lab/config-reference)
- [Referencia de la API de DLE](https://postgres.ai/swagger-ui/dblab/)
- [Referencia CLI del cliente](https://postgres.ai/docs/database-lab/cli-reference)

### Guías prácticas
- [Cómo instalar Database Lab con Terraform en AWS](https://postgres.ai/docs/how-to-guides/administration/install-database-lab-with-terraform)
- [Cómo instalar e inicializar CLI de Database Lab](https://postgres.ai/docs/guides/cli/cli-install-init)
- [Cómo gestionar DLE](https://postgres.ai/docs/how-to-guides/administration)
- [Cómo trabajar con clones](https://postgres.ai/docs/how-to-guides/cloning)

Puede encontrar más en [la sección "Guías prácticas"](https://postgres.ai/docs/how-to-guides) de los documentos.

### Varios
- [Imágenes DLE Docker](https://hub.docker.com/r/postgresai/dblab-server)
- [Imágenes de Docker extendidas para PostgreSQL (con muchas extensiones)] (https://hub.docker.com/r/postgresai/extended-postgres)
- [Chatbot de optimización de SQL (Joe Bot)](https://postgres.ai/docs/joe-bot)
- [Comprobador de migración de base de datos](https://postgres.ai/docs/db-migration-checker)

## Licencia
El código fuente de DLE tiene la licencia de código abierto aprobada por OSI GNU Affero General Public License versión 3 (AGPLv3).

Comuníquese con el equipo de Postgres.ai si desea una licencia comercial o de prueba que no contenga las cláusulas GPL: [Página de contacto](https://postgres.ai/contact).

## Comunidad & Apoyo
- ["Código de conducta del Pacto de la comunidad de motor de laboratorio de base de datos"](../CODE_OF_CONDUCT.md)
- Dónde obtener ayuda: [Página de contacto](https://postgres.ai/contact)
- [Slack de la comunidad](https://slack.postgres.ai)
- Si necesita informar un problema de seguridad, siga las instrucciones en ["Directrices de seguridad del motor de laboratorio de base de datos"](../SECURITY.md).

[![Pacto de colaborador](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?color=blue)](../CODE_OF_CONDUCT.md)
