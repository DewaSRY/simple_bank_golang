# REST Testing with VS Code REST Client

Use this folder to test the API from VS Code using the REST Client extension.

## 1) Install Extension

Install `REST Client` by Huachao Mao.

Extension ID: `humao.rest-client`

## 2) Start the API Server

1. Fill in `app.env` at project root with valid values:
	- `DB_DRIVER`
	- `DB_SOURCE`
	- `SERVER_ADDRESS` (example: `:8080`)
2. Start server:

```bash
make server
```

## 3) Run Requests

1. Open `rest-testing/core-service.http`.
2. Click `Send Request` above a request.
3. Run in this order:
	- `Health`
	- `Create Account A`
	- `Create Account B`
	- `Get Account A`
	- `Transfer A -> B`

The file uses request names and variables so IDs from account creation can be reused automatically.

## 4) Troubleshooting

- `connection refused`: server is not running, or `baseUrl` is wrong.
- `500` on account/transfer: check database connection and migrated schema.
- `400` validation errors: verify request body fields and currency values (`USD`, `EUR`, `GBP`, `IDR`).


