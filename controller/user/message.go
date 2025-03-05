package user

import (
	"fmt"
	"gradspaceBK/database"
	"math/rand"
	"strconv"
	"time"

	"gradspaceBK/middlewares"
	"gradspaceBK/ws"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func RegisterMessageRoutes(base *fiber.Group) {
	msg := base.Group("/messages")
	msg.Get("/conversations/", middlewares.AuthMiddleware, GetConversations)
	msg.Get("/suggested/users", middlewares.AuthMiddleware, GetSuggestedUsers)
	msg.Post("/", middlewares.AuthMiddleware, SendMessage)
	msg.Get("/:otherUserId", middlewares.AuthMiddleware, GetMessages)
	msg.Get("/search/:searchKey", middlewares.AuthMiddleware, GetSearchMatchUsers)
	msg.Post("/conversation/:conversationID/clear/", middlewares.AuthMiddleware, ClearConversation)
}

type MessageData struct {
	RecipientID string `json:"recipientId"`
	Content     string `json:"content"`
}

func SendMessage(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	session := database.Session.Db

	user := database.User{}
	if err := session.Model(&database.User{}).
		Where("id = ?", userData["user_id"].(string)).
		First(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Sender not found",
		})
	}

	var messageData MessageData
	if err := c.BodyParser(&messageData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	var conversation database.Conversation
	var newMessage database.Message

	txErr := session.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("participant1_id = ? AND participant2_id = ?", user.ID, messageData.RecipientID).
			Or("participant1_id = ? AND participant2_id = ?", messageData.RecipientID, user.ID).
			First(&conversation).Error; err != nil {
			conversation = database.Conversation{
				Participant1ID: user.ID,
				Participant2ID: messageData.RecipientID,
			}
			if err := tx.Create(&conversation).Error; err != nil {
				return err
			}
		}
		var receiver_id string
		if user.ID == conversation.Participant1ID {
			receiver_id = conversation.Participant2ID
		} else {
			receiver_id = conversation.Participant1ID
		}
		message := database.Message{
			ConversationID: conversation.ID,
			SenderID:       user.ID,
			ReceiverID:     receiver_id,
			Text:           messageData.Content,
			Seen:           false,
		}
		if err := tx.Create(&message).Error; err != nil {
			return err
		}
		newMessage = message
		if err := tx.Model(&conversation).Updates(map[string]interface{}{
			"last_message":             message.Text,
			"last_message_sender_id":   message.SenderID,
			"last_message_receiver_id": message.ReceiverID,
		}).Error; err != nil {
			return err
		}

		return nil
	})
	if txErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to process message",
		})
	}

	responseData := map[string]interface{}{
		"id":        newMessage.ID,
		"seen":      newMessage.Seen,
		"senderId":  newMessage.SenderID,
		"text":      newMessage.Text,
		"createdAt": newMessage.CreatedAt.Format(time.RFC3339),
	}

	// Send real-time update to recipient
	if recipientConn := ws.GetSocket(messageData.RecipientID); recipientConn != nil {
		realTimeMessage := responseData
		realTimeMessage["conversationId"] = conversation.ID
		if err := recipientConn.WriteJSON(fiber.Map{
			"type":    "NEW_MESSAGE",
			"message": realTimeMessage,
		}); err != nil {
			fmt.Println("Error sending message:", err)
		}
	}

	return c.JSON(responseData)
}

type MessageResponse struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	SenderID  string    `json:"senderId"`
	Seen      bool      `json:"seen"`
	CreatedAt time.Time `json:"createdAt"`
}

func GetMessages(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	session := database.Session.Db
	user := database.User{}
	if err := session.Model(&database.User{}).
		Where("id = ?", userData["user_id"].(string)).
		First(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Sender not found",
		})
	}

	otherUserId := c.Params("otherUserId")

	var conversation database.Conversation

	if err := session.Where("participant1_id = ? AND participant2_id = ?", user.ID, otherUserId).
		Or("participant1_id = ? AND participant2_id = ?", otherUserId, user.ID).
		First(&conversation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Conversation not found",
		})
	}
	messages := []database.Message{}
	if err := session.Where("conversation_id = ? AND (sender_id = ? OR receiver_id = ?)", conversation.ID, user.ID, user.ID).
		Find(&messages).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch messages",
		})
	}

	var messageResponses []MessageResponse
	for _, msg := range messages {
		messageResponses = append(messageResponses, MessageResponse{
			ID:        msg.ID,
			Text:      msg.Text,
			SenderID:  msg.SenderID,
			Seen:      msg.Seen,
			CreatedAt: msg.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"messages":     messageResponses,
		"conversation": conversation,
	})
}

type ConversationResponse struct {
	ID                     string    `json:"id"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
	Participant1ID         string    `json:"participant1Id"`
	Participant1FullName   string    `json:"participant1FullName"`
	Participant1ProfileImg string    `json:"participant1ProfileImg"`
	Participant2ID         string    `json:"participant2Id"`
	Participant2FullName   string    `json:"participant2FullName"`
	Participant2ProfileImg string    `json:"participant2ProfileImg"`
	LastMessage            string    `json:"lastMessage"`
	LastMessageSenderID    string    `json:"lastMessageSenderId"`
	LastMessageReceiverID  string    `json:"lastMessageReceiverId"`
	LastMessageSeen        bool      `json:"lastMessageSeen"`
}

func GetConversations(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	session := database.Session.Db
	currentUserID := userData["user_id"].(string)
	var conversations []database.Conversation
	if err := session.Where("participant1_id = ? OR participant2_id = ?", currentUserID, currentUserID).
		Find(&conversations).Error; err != nil {
		fmt.Println("Error fetching conversations:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch conversations",
		})
	}
	participantIDs := make(map[string]struct{})
	for _, conv := range conversations {
		participantIDs[conv.Participant1ID] = struct{}{}
		participantIDs[conv.Participant2ID] = struct{}{}
	}
	uniqueParticipantIDs := make([]string, 0, len(participantIDs))
	for id := range participantIDs {
		uniqueParticipantIDs = append(uniqueParticipantIDs, id)
	}
	type UserWithProfile struct {
		ID           string `gorm:"column:id"`
		FullName     string `gorm:"column:full_name"`
		ProfileImage string `gorm:"column:profile_image"`
	}
	var users []UserWithProfile
	if len(uniqueParticipantIDs) > 0 {
		if err := session.Table("users").
			Select("users.id, users.full_name, user_profiles.profile_image").
			Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
			Where("users.id IN ?", uniqueParticipantIDs).
			Scan(&users).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to fetch participant details",
			})
		}
	}
	// Create a map of user ID to user details
	userMap := make(map[string]UserWithProfile)
	for _, user := range users {
		userMap[user.ID] = user
	}
	// Build the response with participant details
	response := make([]ConversationResponse, 0, len(conversations))
	for _, conv := range conversations {
		// Get participant 1 details
		participant1, ok1 := userMap[conv.Participant1ID]
		participant1FullName := ""
		participant1ProfileImg := ""
		if ok1 {
			participant1FullName = participant1.FullName
			participant1ProfileImg = participant1.ProfileImage
		}
		// Get participant 2 details
		participant2, ok2 := userMap[conv.Participant2ID]
		participant2FullName := ""
		participant2ProfileImg := ""
		if ok2 {
			participant2FullName = participant2.FullName
			participant2ProfileImg = participant2.ProfileImage
		}
		var lastMessage string = ""
		if conv.LastMessageReceiverID == currentUserID || conv.LastMessageSenderID == currentUserID {
			lastMessage = conv.LastMessage
		}

		response = append(response, ConversationResponse{
			ID:                     conv.ID,
			CreatedAt:              conv.CreatedAt,
			UpdatedAt:              conv.UpdatedAt,
			Participant1ID:         conv.Participant1ID,
			Participant1FullName:   participant1FullName,
			Participant1ProfileImg: participant1ProfileImg,
			Participant2ID:         conv.Participant2ID,
			Participant2FullName:   participant2FullName,
			Participant2ProfileImg: participant2ProfileImg,
			LastMessage:            lastMessage,
			LastMessageSenderID:    conv.LastMessageSenderID,
			LastMessageReceiverID:  conv.LastMessageReceiverID,
			LastMessageSeen:        conv.LastMessageSeen,
		})
	}
	return c.JSON(fiber.Map{
		"success":       true,
		"conversations": response,
	})
}

func GetSuggestedUsers(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	session := database.Session.Db
	currentUserID := userData["user_id"].(string)

	// Fetch users the current user is following (limit 10)
	var following []database.Follow
	if err := session.Where("follower_id = ?", currentUserID).Limit(10).Find(&following).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch following users",
		})
	}
	followingIDs := make([]string, len(following))
	for i, f := range following {
		followingIDs[i] = f.FollowingID
	}

	// Fetch recent conversation participants (limit 10)
	var conversations []database.Conversation
	if err := session.Where("participant1_id = ? OR participant2_id = ?", currentUserID, currentUserID).
		Order("updated_at DESC").
		Limit(10).
		Find(&conversations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch recent conversations",
		})
	}
	var participantIDs []string
	for _, conv := range conversations {
		if conv.Participant1ID == currentUserID {
			participantIDs = append(participantIDs, conv.Participant2ID)
		} else {
			participantIDs = append(participantIDs, conv.Participant1ID)
		}
	}

	// Combine, deduplicate, and exclude current user
	allIDs := append(followingIDs, participantIDs...)
	seen := make(map[string]bool)
	var uniqueIDs []string
	for _, id := range allIDs {
		if id != currentUserID && !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	// Shuffle for variety
	rand.Shuffle(len(uniqueIDs), func(i, j int) {
		uniqueIDs[i], uniqueIDs[j] = uniqueIDs[j], uniqueIDs[i]
	})

	// Pagination setup
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit
	totalItems := len(uniqueIDs)
	totalPages := (totalItems + limit - 1) / limit

	// Apply pagination
	start := offset
	end := offset + limit
	if start > totalItems {
		start = totalItems
	}
	if end > totalItems {
		end = totalItems
	}
	paginatedIDs := uniqueIDs[start:end]

	// Fetch user details with LATEST profile (critical fix)
	type UserWithProfile struct {
		ID           string `gorm:"column:id"`
		FullName     string `gorm:"column:full_name"`
		ProfileImage string `gorm:"column:profile_image"`
	}
	var users []UserWithProfile
	if len(paginatedIDs) > 0 {
		// Get latest profile for each user
		subQuery := session.Table("user_profiles").
			Select("DISTINCT ON (user_id) user_id, profile_image").
			Order("user_id, created_at DESC")

		if err := session.Table("users").
			Select("users.id, users.full_name, up.profile_image").
			Joins("LEFT JOIN (?) AS up ON users.id = up.user_id", subQuery).
			Where("users.id IN ?", paginatedIDs).
			Scan(&users).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to fetch user details",
			})
		}
	}

	// Build response
	response := make([]fiber.Map, len(users))
	for i, user := range users {
		response[i] = fiber.Map{
			"recipientId":         user.ID,
			"recipientFullName":   user.FullName,
			"recipientProfileImg": user.ProfileImage,
		}
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"recipents": response,
		"meta": fiber.Map{
			"current_page": page,
			"per_page":     limit,
			"total_pages":  totalPages,
			"total_items":  totalItems,
		},
	})
}

func GetSearchMatchUsers(c *fiber.Ctx) error {
	searchKey := c.Params("searchKey")
	session := database.Session.Db
	userData := c.Locals("user_data").(jwt.MapClaims)
	currentUserID := userData["user_id"].(string)

	if searchKey == "" {
		return c.JSON(fiber.Map{
			"success": true,
			"users":   []interface{}{},
			"meta":    fiber.Map{"total": 0},
		})
	}

	// Pagination setup
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Base query for search
	query := session.Table("users").
		Select("users.id, users.full_name, user_profiles.profile_image").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("users.is_onboard = ?", true).
		Where("users.id != ?", currentUserID). // Exclude current user
		Where("(LOWER(COALESCE(users.user_name, '')) LIKE LOWER(?) OR LOWER(users.full_name) LIKE LOWER(?))",
			"%"+searchKey+"%", "%"+searchKey+"%")

	// Count total matches first
	var totalItems int64
	if err := query.Model(&database.User{}).Count(&totalItems).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to count results",
		})
	}

	// Get paginated results
	var users []struct {
		ID           string `gorm:"column:id"`
		FullName     string `gorm:"column:full_name"`
		ProfileImage string `gorm:"column:profile_image"`
	}

	if err := query.
		Order("users.full_name ASC"). // Sort by name
		Offset(offset).
		Limit(limit).
		Scan(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch users",
		})
	}

	totalPages := (int(totalItems) + limit - 1) / limit
	response := make([]fiber.Map, len(users))
	for i, user := range users {
		response[i] = fiber.Map{
			"recipientId":         user.ID,
			"recipientFullName":   user.FullName,
			"recipientProfileImg": user.ProfileImage,
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"users":   response,
		"meta": fiber.Map{
			"current_page": page,
			"per_page":     limit,
			"total_pages":  totalPages,
			"total_items":  totalItems,
		},
	})
}

func ClearConversation(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	currentUserID := userData["user_id"].(string)
	conversationID := c.Params("conversationID")
	var conversation database.Conversation
	session := database.Session.Db

	if err := session.Model(&database.Conversation{}).
		Where("id = ? AND (participant1_id = ? OR participant2_id = ?)", conversationID, currentUserID, currentUserID).
		First(&conversation).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Conversation not found",
		})
	}
	session.Model(&database.Message{}).Where("conversation_id = ? AND receiver_id = ?",
		conversationID, currentUserID).UpdateColumn("receiver_id", nil)
	session.Model(&database.Message{}).Where("conversation_id = ? AND sender_id = ?",
		conversationID, currentUserID).UpdateColumn("sender_id", nil)
	if conversation.LastMessageReceiverID == currentUserID {
		log.Info("receiver")
		session.Model(&database.Conversation{}).Where("id = ?",
			conversationID).UpdateColumn("last_message_receiver_id", nil)
	}
	if conversation.LastMessageSenderID == currentUserID {
		log.Info("sender")
		session.Model(&database.Conversation{}).Where("id = ?",
			conversationID).UpdateColumn("last_message_sender_id", nil)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Successfully cleared the chat",
	})
}
