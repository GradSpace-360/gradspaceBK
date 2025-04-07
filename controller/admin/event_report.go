package admin

import (
	"gradspaceBK/database"
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

type PosterResponse struct {
		ID           string `json:"id"`
		FullName     string `json:"full_name"`
		UserName     string `json:"user_name"`
		ProfileImage string `json:"profile_image"`
}

type EventResponse struct {
		ID                 string         `json:"id"`
		Title              string         `json:"title"`
		Description        string         `json:"description"`
		Venue              string         `json:"venue"`
		EventType          string         `json:"event_type"`
		RegisterLink       string         `json:"register_link"`
		StartDateTime      time.Time      `json:"start_date_time"`
		EndDateTime        time.Time      `json:"end_date_time"`
		IsRegistrationOpen bool           `json:"is_registration_open"`
		PostedBy           PosterResponse `json:"posted_by"`
		CreatedAt          string         `json:"created_at"`
		IsSaved            bool           `json:"is_saved"`
}

func AdminEventRoutes(base *fiber.Group) {
	admin := base.Group("/admin")
	admin.Get("/event-reports", GetEventReports)
	admin.Get("/event-reports/:eventId", GetEventReportDetails)
	admin.Delete("/event-reports/:eventId/dismiss", DismissEventReports)
	admin.Delete("/events/:eventId", AdminDeleteEvent)
}

func GetEventReports(c *fiber.Ctx) error {
    // Pagination parameters
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit, _ := strconv.Atoi(c.Query("limit", "10"))
    
    // Validate pagination parameters
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 10
    }
    offset := (page - 1) * limit

    type ReportSummary struct {
        EventID      string    `json:"event_id"`
        EventType    string    `json:"event_type"`
        Title        string    `json:"title"`
        PostedBy     string    `json:"-"`
        UserFullName string   `json:"-"`
        FlagCount    int      `json:"flag_count"`
        TopReason    string   `json:"top_reason"`
        LastFlagged  time.Time `json:"-"`
        PostDate     time.Time `json:"-"`
    }

    var summaries []ReportSummary
    query := `
        SELECT 
            e.id AS event_id,
            e.event_type AS event_type,
            e.title AS title,
            e.posted_by AS posted_by,
            u.full_name AS user_full_name,
            COUNT(er.id) AS flag_count,
            (SELECT reason 
             FROM event_reports 
             WHERE event_id = e.id 
             GROUP BY reason 
             ORDER BY COUNT(*) DESC 
             LIMIT 1) AS top_reason,
            MAX(er.created_at) AS last_flagged,
            e.created_at AS post_date
        FROM event_reports er
        JOIN events e ON er.event_id = e.id
        JOIN users u ON e.posted_by = u.id
        GROUP BY e.id, u.full_name, e.created_at
        ORDER BY last_flagged DESC
        LIMIT ? OFFSET ?
    `

    if err := database.Session.Db.Raw(query, limit, offset).Scan(&summaries).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to fetch reports",
        })
    }

    response := make([]fiber.Map, 0)
    for _, s := range summaries {
        response = append(response, fiber.Map{
            "event_id":     s.EventID,
            "event_type":   s.EventType,
            "title":        s.Title,    
            "posted_by":    fiber.Map{"id": s.PostedBy, "name": s.UserFullName},
            "flag_count":   s.FlagCount,
            "top_reason":   s.TopReason,
            "last_flagged": s.LastFlagged.Format(time.RFC3339),
            "post_date":    s.PostDate.Format("2006-01-02"),
        })
    }

    // Get total count of distinct reported events
    var total int64
    database.Session.Db.Model(&database.EventReport{}).
        Select("COUNT(DISTINCT event_id)").
        Scan(&total)

    return c.JSON(fiber.Map{
        "success": true,
        "data":    response,
        "meta": fiber.Map{
            "current_page": page,
            "per_page":     limit,
            "total_pages":  int(math.Ceil(float64(total) / float64(limit))),
            "total_items":  total,
        },
    })
}

func GetEventReportDetails(c *fiber.Ctx) error {
	eventID := c.Params("eventId")

	var event database.Event
	if err := database.Session.Db.
		Preload("User").
		Where("id = ?", eventID).
		First(&event).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Event not found",
		})
	}

	var reports []database.EventReport
	if err := database.Session.Db.
		Preload("Reporter").
		Where("event_id = ?", eventID).
		Find(&reports).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch reports",
		})
	}

	// Fetch user profile for poster
	var userProfile database.UserProfile
	database.Session.Db.Where("user_id = ?", event.User.ID).First(&userProfile)

	userName := ""
	if event.User.UserName != nil {
		userName = *event.User.UserName
	}

	poster := PosterResponse{
		ID:           event.User.ID,
		FullName:     event.User.FullName,
		UserName:     userName,
		ProfileImage: userProfile.ProfileImage,
	}

	eventResponse := EventResponse{
		ID:                 event.ID,
		Title:              event.Title,
		Description:        event.Description,
		Venue:              event.Venue,
		EventType:          string(event.EventType),
		RegisterLink:       event.RegisterLink,
		StartDateTime:      event.StartDateTime,
		EndDateTime:        event.EndDateTime,
		IsRegistrationOpen: event.IsRegistrationOpen,
		PostedBy:           poster,
		CreatedAt:          event.CreatedAt.Format("2006-01-02"),
		IsSaved:            false,
	}

	flagData := make([]fiber.Map, 0)
	for _, r := range reports {
		flagData = append(flagData, fiber.Map{
			"person_who_flagged": fiber.Map{
				"id":   r.ReporterID,
				"name": r.Reporter.FullName,
			},
			"reason":    r.Reason,
			"flag_date": r.CreatedAt.Format(time.RFC3339),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"event_data": eventResponse,
			"flag_data": flagData,
		},
	})
}
func DismissEventReports(c *fiber.Ctx) error {
	eventID := c.Params("eventId")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid event ID",
		})
	}

	// Check if there are any reports for the given event
	var count int64
	if err := database.Session.Db.Model(&database.EventReport{}).
		Where("event_id = ?", eventID).
		Count(&count).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to check reports",
		})
	}

	if count == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "No reports found for the given event",
		})
	}

	// Dismiss reports using a transaction for safety
	tx := database.Session.Db.Begin()
	if err := tx.Where("event_id = ?", eventID).Delete(&database.EventReport{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to dismiss reports",
		})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to commit dismissal",
		})
	}

	return c.JSON(fiber.Map{
		"success":         true,
		"message":         "Reports dismissed successfully",
		"dismissed_count": count,
	})
}
func AdminDeleteEvent(c *fiber.Ctx) error {
	eventID := c.Params("eventId")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid event ID",
		})
	}

	// Begin transaction
	tx := database.Session.Db.Begin()
	// Ensure proper rollback in case of panic.
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete the event record.
	if err := tx.Where("id = ?", eventID).Delete(&database.Event{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to delete event",
		})
	}

	// Cleanup related event reports.
	if err := tx.Where("event_id = ?", eventID).Delete(&database.EventReport{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to cleanup event reports",
		})
	}

	// Commit the transaction.
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Transaction commit failed",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Event deleted successfully",
	})
}
