# Normalize Response

This document describes the standard response format used by the REST API. All API endpoints should follow this structure to ensure responses are consistent and predictable.

## Success Response

A successful request should return the following structure:

```json id="z9m2qd"
{
  "data": {},
  "message": "Success"
}
```

### Example

```json id="q7nxt1"
{
  "data": {
    "id": "22bbacb9-848c-4011-a5ed-f9d2b3320901",
    "name": "Example Data"
  },
  "message": "Data retrieved successfully"
}
```

## Error Response

An error response should return the following structure:

```json id="skmf0z"
{
  "error": {
    "code": "ERROR_CODE",
    "message": "An error occurred",
    "details": []
  }
}
```

The `details` field is optional and should be used when the error is related to one or more specific fields.

### Field Error Structure

Each field error should contain information about the field that caused the error:

```json id="v4bqyt"
{
  "field": "field_name",
  "message": "Description of the error"
}
```

### Example: Duplicate Name

For example, if the request contains a `name` field and the database has a unique constraint that prevents duplicate names:

```json id="brd2j7"
{
  "error": {
    "code": "DUPLICATE_RESOURCE",
    "message": "The request could not be completed because some data already exists",
    "details": [
      {
        "field": "name",
        "message": "Name already exists"
      }
    ]
  }
}
```

### Example: Multiple Field Errors

If multiple fields contain invalid values, all field errors can be returned together:

```json id="e4g4px"
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "One or more fields are invalid",
    "details": [
      {
        "field": "name",
        "message": "Name cannot be empty"
      },
      {
        "field": "email",
        "message": "Email format is invalid"
      }
    ]
  }
}
```

## Pagination Response

For endpoints that return paginated data, the response should include pagination metadata.

```json id="ryxxfu"
{
  "data": [],
  "message": "Data retrieved successfully",
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 100
  }
}
```

## Response Guidelines

- All successful responses should contain a `data` field.
- All error responses should contain an `error` field.
- Use `details` only when an error is related to specific request fields.
- Each item in `details` should clearly identify the field and the reason for the error.
- Use consistent error codes throughout the API.
- Pagination information should be placed inside the `meta` field.
- Do not expose raw database or internal server errors directly to API consumers.

## Summary

The normalized response format provides a consistent structure for both successful and failed requests. For field-specific errors, the `details` array allows API consumers, such as frontend applications, to easily identify which fields need to be corrected.
