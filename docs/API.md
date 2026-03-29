# API Documentation

> API documentation — generated from Swagger annotations

## Overview

The GeoCore Next API is a RESTful API built with Go and the Gin framework. All endpoints return JSON responses.

## Base URL

```
Development: http://localhost:8080/api/v1
Production: https://api.yourdomain.com/api/v1
```

## Authentication

Most endpoints require a JWT token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

## Swagger UI

When running locally, access the interactive API documentation at:

```
http://localhost:8080/api/docs
```

## Rate Limiting

- **Authenticated users**: 100 requests per minute
- **Unauthenticated users**: 20 requests per minute

Rate limit headers are included in all responses:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

## Error Responses

All errors follow this format:

```json
{
  "error": "error_code",
  "message": "Human readable error message"
}
```

## Endpoints

See the main [README.md](../README.md) for a quick reference of available endpoints.

For detailed request/response schemas, use the Swagger UI when running locally.
