# Database Lab Engine API

## Directory Contents
- `swagger-spec` – OpenAPI 3.0 specification of DBLab API
- `swagger-ui` – Swagger UI to see the API specification (embedded in DBLab, available at :2345 or :2346/api)
- `postman` – [Postman](https://www.postman.com/) collection and environment files, used to test API in CI/CD pipelines (running [`newman`](https://github.com/postmanlabs/newman))

## API Documentation

Detailed information about API design principles and testing has been moved to the main [CONTRIBUTING.md](../../CONTRIBUTING.md#api-design-and-testing) file. Please refer to that document for:

- API design principles
- API documentation workflow
- Testing with Postman and Newman
- CI/CD integration for API tests

This centralized approach ensures all development information is maintained in one place.

## Quick Reference

### API Documentation Sites
- API documentation is hosted at https://dblab.readme.io/ and https://api.dblab.dev
- The OpenAPI specification in this directory is the source of truth for the API

### Generating Postman Collection
```
portman --cliOptionsFile engine/api/postman/portman-cli.json
```