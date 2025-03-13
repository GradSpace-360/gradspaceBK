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

## Event Endpoints

### `GET /events/`

**Description**  
Retrieve a list of events with optional filtering and pagination. Includes details about whether the event is saved by the authenticated user.

**Authentication**  
Required

**Query Parameters**
| Parameter | Type | Required | Default | Description |
|-----------|--------|----------|---------|-------------|
| `search` | string | No | - | Filter by event title. |
| `venue` | string | No | - | Filter by event venue. |
| `event_type` | string | No | - | Filter by event type (e.g., `ALUM_EVENT`). |
| `start_date` | date | No | - | Filter events starting on or after this date (ISO 8601 format). |
| `page` | integer | No | 1 | Page number. |
| `limit` | integer | No | 10 | Number of records per page. |

**Response Format**

```json
{
  "success": true,
  "data": {
    "events": [
      {
        "id": "string",
        "title": "string",
        "description": "string",
        "venue": "string",
        "event_type": "string",
        "register_link": "string (url)",
        "start_date_time": "ISO 8601 timestamp",
        "end_date_time": "ISO 8601 timestamp",
        "is_registration_open": true,
        "posted_by": {
          "id": "string",
          "full_name": "string",
          "username": "string",
          "profile_image": "string"
        },
        "created_at": "date (YYYY-MM-DD)",
        "is_saved": true
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

**Sample Response**

```json
{
  "success": true,
  "data": {
    "events": [
      {
        "id": "aa6d5d3e-1f17-4304-a15e-61138ea4a2e1",
        "title": "Mech Alumni Industry Connect 2025",
        "description": "# Mechanical Engineering Industry Symposium...",
        "venue": "College of Engineering Adoor...",
        "event_type": "ALUM_EVENT",
        "register_link": "",
        "start_date_time": "2025-03-01T10:00:00+05:30",
        "end_date_time": "2025-03-03T17:00:00+05:30",
        "is_registration_open": true,
        "posted_by": {
          "id": "dd9a8d18-8156-4633-8b48-19a43c20724d",
          "full_name": "Arjun Menon",
          "username": "arjun_menon",
          "profile_image": "uploads/profile/dd9a8d18-8156-4633-8b48-19a43c20724d.jpg"
        },
        "created_at": "2025-03-12",
        "is_saved": false
      }
    ],
    "pagination": {
      "total": 2,
      "page": 1,
      "limit": 10
    }
  }
}
```

---

### `GET /events/saved`

**Description**  
Retrieve a list of events saved by the authenticated user.

**Authentication**  
Required

**Query Parameters**
| Parameter | Type | Required | Default |
|-----------|---------|----------|---------|
| `page` | integer | No | 1 |
| `limit` | integer | No | 10 |

**Response Format**  
Same structure as `GET /events/` but lists only saved events.

**Sample Response**

```json
{
  "success": true,
  "data": {
    "events": [],
    "pagination": {
      "total": 0,
      "page": 1,
      "limit": 10
    }
  }
}
```

---

### `POST /events/save`

**Description**  
Toggle saving/unsaving an event for the authenticated user.

**Authentication**  
Required

**Request Body**

```json
{
  "event_id": "string"
}
```

**Response Format**

```json
{
  "success": true,
  "action": "saved" // or "removed"
}
```

**Sample Request**

```json
{
  "event_id": "a2dff955-bce3-49fd-b4c2-188936e60661"
}
```

**Sample Response**

```json
{
  "success": true,
  "action": "saved"
}
```

---

### `GET /events/my-events`

**Description**  
Retrieve events posted by the authenticated user.

**Authentication**  
Required

**Query Parameters**
| Parameter | Type | Required | Default |
|-----------|---------|----------|---------|
| `page` | integer | No | 1 |
| `limit` | integer | No | 10 |

**Response Format**  
Same structure as `GET /events/` but lists only the user’s posted events.

**Sample Response**

```json
{
  "success": true,
  "data": {
    "events": [
      {
        "id": "1696f74c-0920-402d-b43d-9bf577b23fc0",
        "title": "CSE Alumni Tech Summit 2024",
        "description": "# Annual CSE Alumni Meet...",
        "venue": "College of Engineering Adoor...",
        "event_type": "ALUM_EVENT",
        "register_link": "https://alum.coeadoor.edu.in/register/cse-summit",
        "start_date_time": "2024-10-12T09:00:00+05:30",
        "end_date_time": "2024-10-13T20:00:00+05:30",
        "is_registration_open": false,
        "posted_by": {
          "id": "fbdffa0e-f848-4fb7-aed5-faaf6b8ba906",
          "full_name": "Maria George",
          "username": "maria_george",
          "profile_image": "uploads/profile/fbdffa0e-f848-4fb7-aed5-faaf6b8ba906.jpg"
        },
        "created_at": "2025-03-12",
        "is_saved": false
      }
    ],
    "pagination": {
      "total": 1,
      "page": 1,
      "limit": 10
    }
  }
}
```

---

### `POST /events/`

**Description**  
Create a new event listing.

**Authentication**  
Required

**Request Body**

```json
{
  "title": "string (min 5, max 255)",
  "description": "string (min 20)",
  "venue": "string",
  "event_type": "string (e.g., ALUM_EVENT)",
  "register_link": "string (optional, valid URL)",
  "start_date_time": "ISO 8601 timestamp",
  "end_date_time": "ISO 8601 timestamp"
}
```

**Response Format**

```json
{
  "success": true,
  "data": {
    "id": "string",
    "title": "string",
    "venue": "string",
    "event_type": "string",
    "start_time": "ISO 8601 timestamp"
  }
}
```

**Sample Request**

```json
{
  "title": "CSE Alumni Tech Summit 2024",
  "description": "# Annual CSE Alumni Meet...",
  "venue": "College of Engineering Adoor...",
  "event_type": "ALUM_EVENT",
  "register_link": "https://alum.coeadoor.edu.in/register/cse-summit",
  "start_date_time": "2024-10-12T09:00:00+05:30",
  "end_date_time": "2024-10-13T20:00:00+05:30"
}
```

**Sample Response**

```json
{
  "success": true,
  "data": {
    "id": "1696f74c-0920-402d-b43d-9bf577b23fc0",
    "title": "CSE Alumni Tech Summit 2024",
    "venue": "College of Engineering Adoor...",
    "event_type": "ALUM_EVENT",
    "start_time": "2024-10-12T09:00:00+05:30"
  }
}
```

---

### `PATCH /events/:id/status`

**Description**  
Update the registration status of an event (open/closed).

**Authentication**  
Required

**URL Parameter**
| Parameter | Type | Description |
|-----------|--------|---------------------|
| `id` | string | Event identifier |

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

**Sample Request**

```json
{
  "is_open": false
}
```

**Sample Response**

```json
{
  "success": true,
  "is_open": false
}
```

---

### `DELETE /events/:id`

**Description**  
Delete an event posted by the authenticated user.

**Authentication**  
Required

**URL Parameter**
| Parameter | Type | Description |
|-----------|--------|---------------------|
| `id` | string | Event identifier |

**Response Format**

```json
{
  "success": true
}
```
