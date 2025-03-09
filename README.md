# API Documentation

## Job Endpoints

### `GET /jobs/`

**Description**  
Retrieve a list of job listings with optional filtering and pagination.

**Authentication**  
Optional

**Query Parameters**
| Parameter | Type | Required | Default | Description |
|--------------|----------|----------|---------|---------------------------------|
| `search` | string | No | - | Filter by job title. |
| `location` | string | No | - | Filter by job location. |
| `company_id` | string | No | - | Filter by company identifier. |
| `page` | integer | No | 1 | Page number. |
| `limit` | integer | No | 10 | Number of records per page. |

**Response Format**

```json
{
  "success": true,
  "data": {
    "jobs": [
      {
        "id": "string",
        "title": "string",
        "description": "string",
        "location": "string",
        "requirements": "string",
        "is_open": true,
        "job_type": "Part-Time | Full-Time | Internship | Freelance",
        "apply_link": "string (url)",
        "company": {
          "name": "string",
          "logo_url": "string"
        },
        "posted_by": {
          "id": "string",
          "full_name": "string",
          "username": "string",
          "profile_image": "string"
        }
      }
    ],
    "pagination": {
      "total": 0,
      "page": 1,
      "limit": 10
    }
  }
}
```

---

### `GET /jobs/saved`

**Description**  
Retrieve a list of jobs saved by the authenticated user.

**Authentication**  
Required

**Query Parameters**
| Parameter | Type | Required | Default |
|-----------|---------|----------|---------|
| `page` | integer | No | 1 |
| `limit` | integer | No | 10 |

**Response Format**  
Same as `GET /jobs/` but returns only saved jobs.

---

### `POST /jobs/save`

**Description**  
Toggle saving a job for the authenticated user.

**Authentication**  
Required

**Request Body**

```json
{
  "job_id": "string"
}
```

**Response Format**

```json
{
  "success": true,
  "action": "saved" // or "removed"
}
```

---

### `GET /jobs/my-jobs`

**Description**  
Retrieve jobs posted by the authenticated user.

**Authentication**  
Required

**Query Parameters**
| Parameter | Type | Required | Default |
|-----------|---------|----------|---------|
| `page` | integer | No | 1 |
| `limit` | integer | No | 10 |

**Response Format**  
Same structure as `GET /jobs/` but listing only the user’s own job postings.

---

### `POST /jobs/`

**Description**  
Create a new job listing.

**Authentication**  
Required

**Request Body**

```json
{
  "title": "string (min 5, max 255)",
  "company_id": "string (uuid)",
  "description": "string (min 20)",
  "location": "string",
  "requirements": "string",
  "job_type": "Part-Time | Full-Time | Internship | Freelance",
  "apply_link": "string (optional, valid URL)"
}
```

**Response Format**

```json
{
  "success": true,
  "data": {
    "id": "string",
    "title": "string",
    "company": "string",
    "location": "string",
    "created_at": "timestamp"
  }
}
```

---

### `PATCH /jobs/:id/status`

**Description**  
Update the hiring status of a job posting.

**Authentication**  
Required

**URL Parameter**
| Parameter | Type | Description |
|-----------|--------|---------------------|
| `id` | string | Job identifier |

**Request Body**

```json
{
  "is_open": true
}
```

**Response Format**

```json
{
  "success": true,
  "is_open": true
}
```

---

### `DELETE /jobs/:id`

**Description**  
Delete a job listing posted by the authenticated user.

**Authentication**  
Required

**URL Parameter**
| Parameter | Type | Description |
|-----------|--------|---------------------|
| `id` | string | Job identifier |

**Response Format**

```json
{
  "success": true
}
```

---

# Company API Documentation

## Base URL

`/api/companies`

---

### `GET /`

**Description**  
Retrieves a list of all companies in descending chronological order (newest first).

**Response**
**Success (200 OK)**

```json
{
  "data": [
    {
      "ID": "b43563d6-d488-4575-9a13-060eb99c19ce",
      "CreatedAt": "2025-03-09T15:13:42.898047+05:30",
      "UpdatedAt": "2025-03-09T15:25:34.460516+05:30",
      "Name": "Google",
      "LogoURL": "uploads/company-logos/1741531729015807529-c45046e9-643e-488a-b9a1-dbd31a0edfc2.svg"
    }
  ],
  "success": true
}
```

**Error (500 Internal Server Error)**

```json
{
  "error": "Failed to fetch companies"
}
```

---

### `POST /`

**Description**  
Creates a new company. Requires authentication.

**Headers**

- `Content-Type`: `multipart/form-data`
- `Authorization`: Cookies

**Request Body**
| Field | Type | Required | Description |
|-------|--------|----------|---------------------------|
| name | string | Yes | Name of the company |
| logo | file | Yes | Company logo (image file) |

**Response**
**Success (201 Created)**

```json
{
  "data": {
    "ID": "57aca4d4-2573-4c65-a4cf-d6ecb9b3c5c2",
    "CreatedAt": "2025-03-09T19:19:52.207863961+05:30",
    "UpdatedAt": "2025-03-09T19:19:52.207864019+05:30",
    "Name": "suzuki",
    "LogoURL": "uploads/company-logos/1741528192207733841-eed8fc4d-e6cb-49eb-8cb6-bd7c406226de.svg"
  },
  "success": true
}
```

**Errors**
| Status | Response | Condition |
|--------|---------------------------------------|-------------------------------|
| 400 | `{"error": "Company name is required"}` | Missing `name` field |
| 400 | `{"error": "Logo file is required"}` | Missing `logo` file |
| 500 | `{"error": "Failed to create company"}` | Database/file system error |

---

### `PUT /:id`

**Description**  
Updates an existing company. Requires authentication. Supports partial updates.

**URL Parameter**
| Parameter | Type | Required | Description |
|-----------|--------|----------|-------------------|
| id | UUID | Yes | Company ID to update |

**Request Body**
| Field | Type | Required | Description |
|-------|--------|----------|---------------------------|
| name | string | No | New company name |
| logo | file | No | New company logo |

**Response**
**Success (200 OK)**

```json
{
  "data": {
    "ID": "b43563d6-d488-4575-9a13-060eb99c19ce",
    "CreatedAt": "2025-03-09T15:13:42.898047+05:30",
    "UpdatedAt": "2025-03-09T20:18:49.015934183+05:30",
    "Name": "Google",
    "LogoURL": "uploads/company-logos/1741531729015807529-c45046e9-643e-488a-b9a1-dbd31a0edfc2.svg"
  },
  "success": true
}
```

**Errors**
| Status | Response | Condition |
|--------|---------------------------------------|-------------------------------|
| 404 | `{"error": "Company not found"}` | Invalid company ID |
| 500 | `{"error": "Failed to update company"}` | Database/file system error |

---

### `DELETE /:id`

**Description**  
Deletes a company and its associated logo file. Requires authentication.

**URL Parameter**
| Parameter | Type | Required | Description |
|-----------|--------|----------|-------------------|
| id | UUID | Yes | Company ID to delete |

**Response**
**Success (200 OK)**

```json
{
  "message": "Company deleted successfully",
  "success": true
}
```

**Errors**
| Status | Response | Condition |
|--------|---------------------------------------|-------------------------------|
| 404 | `{"error": "Company not found"}` | Invalid company ID |
| 500 | `{"error": "Failed to delete company"}` | Database/file system error |
