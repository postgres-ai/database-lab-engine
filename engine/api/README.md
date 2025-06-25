# Database Lab Engine API

## Directory Contents
- `swagger-spec` – OpenAPI 3.0 specification of DBLab API
- `swagger-ui` – Swagger UI to see the API specification (embedded in DBLab, available at :2345 or :2346/api)
- `postman` – [Postman](https://www.postman.com/) collection and environment files used to test the API in CI/CD pipelines via [`newman`](https://github.com/postmanlabs/newman)

## Design principles
Work in progress: https://gitlab.com/postgres-ai/database-lab/-/merge_requests/744

## API docs
We use ReadMe.io to host the API documentation: https://dblab.readme.io/. Once a new API spec is ready, upload it as a new documentation version and publish.

## Postman, newman, and CI/CD tests
The Postman collection is generated from the OpenAPI spec file using [Portman](https://github.com/apideck-libraries/portman).
1. Install and initialize `portman`.
1. Generate a new version of the Postman collection:
    ```
    portman --cliOptionsFile engine/api/postman/portman-cli.json
    ```
1. Review and adjust the collection:
    - Ensure object creation occurs before its deletion and pass the new object's ID between requests (TODO: provide example).
    - Review and update tests as needed (TODO: details).
1. Commit, push, and ensure Newman's CI/CD testing passes.