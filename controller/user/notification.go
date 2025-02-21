package user

import (
	"gradspaceBK/database"
	"gradspaceBK/middlewares"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func NotificationRoutes(base *fiber.Group) error {
	notifications := base.Group("/notifications")
	notifications.Use(middlewares.AuthMiddleware)
	
	notifications.Get("/", GetNotifications)
	notifications.Post("/read", MarkAsRead)
	
	return nil
}

func GetNotifications(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)

	var notifications []database.Notification
	result := database.Session.Db.
		Preload("Creator").
		Preload("Post").
		Preload("Comment").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&notifications)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch notifications",
		})
	}

	var response []map[string]interface{}
	for _, notification := range notifications {
		notificationData := map[string]interface{}{
			"id":        notification.ID,
			"type":      notification.Type,
			"read":      notification.Read,
			"createdAt": notification.CreatedAt,
			"creator": map[string]interface{}{
				"id":       notification.CreatorID,
				"username": getUsername(notification.CreatorID),
				"image":    getProfileImage(notification.CreatorID),
			},
		}

		if notification.PostID != nil {
			notificationData["post"] = map[string]interface{}{
				"id":      notification.Post.ID,
				"content": notification.Post.Content,
				"image":   notification.Post.Image,
			}
		}

		if notification.CommentID != nil {
			notificationData["comment"] = map[string]interface{}{
				"id":        notification.Comment.ID,
				"content":   notification.Comment.Content,
				"createdAt": notification.Comment.CreatedAt,
			}
		}

		response = append(response, notificationData)
	}

	return c.JSON(response)
}

func MarkAsRead(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)

	type ReadRequest struct {
		NotificationIDs []string `json:"notificationIds"`
	}

	var req ReadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result := database.Session.Db.
		Model(&database.Notification{}).
		Where("user_id = ? AND id IN ?", userID, req.NotificationIDs).
		Update("read", true)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update notifications",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"updated": result.RowsAffected,
	})
}
