## In this directory
- `swagger-spec` – OpenAPI 3.0 specification of DBLab API
- `swagger-ui` – Swagger UI to see the API specification (embedded in DBLab, available at :2345 or :2346/api)
- `postman` – [Postman](https://www.postman.com/) collection and environment files, used to test API in CI/CD pipelines (running [`newman`](https://github.com/postmanlabs/newman))

## Design principles
WIP: https://gitlab.com/postgres-ai/database-lab/-/merge_requests/744

## API docs
We use readme.io to host the API docs: https://dblab.readme.io/. Once a new API spec is ready, upload it there as a new documentation version, and publish.

## Postman, newman, and CI/CD tests
Postman collection is to be generated based on the OpenAPI spec file, using [Portman](https://github.com/apideck-libraries/portman).
1. First, install and initialize `porman`
1. Next, generate a new version of the Postman collection file:
    ```
    portman --cliOptionsFile engine/api/postman/portman-cli.json
    ```
1. Review it, edit, adjust:
    - Object creation first, then deletion of this object, passing the ID of new object from one action to another (TODO: show how)
    - Review and fix tests (TODO: details)
1. Commit, push, ensure `newman` testing works in CI/CD