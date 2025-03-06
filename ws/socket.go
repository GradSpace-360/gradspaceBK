package ws

import (
	"encoding/json"
	"gradspaceBK/database"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"gorm.io/gorm"
)

var connectedUsers = make(map[string]*websocket.Conn)
var mu sync.Mutex

// getSocket returns the websocket connection for the given recipientId.
// It returns nil if no connection exists.
func GetSocket(recipientId string) *websocket.Conn {
	mu.Lock()
	defer mu.Unlock()
	return connectedUsers[recipientId]
}


// BaseMessage is the base structure for all messages
// received by the WebSocket server.
type BaseMessage struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

type MarkMessagesAsSeenPayload struct {
    ConversationID string `json:"conversationId"`
    UserID         string `json:"userId"`
}

type TextMessagePayload struct {
    Text           string `json:"text"`
    SenderID       string `json:"senderId"`
    RecipientID    string `json:"recipientId"`
    ConversationID string `json:"conversationId"`
}


func SetupWebSocket(app *fiber.App) {
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(conn *websocket.Conn) {
		var userID string
		defer func() {
			mu.Lock()
			delete(connectedUsers, userID)
			mu.Unlock()
			conn.Close()
			log.Printf("User disconnected: %s", userID)
			broadcastOnlineUsers() // Broadcast after user disconnects
		}()

		userID = conn.Query("userId")
		if userID == "" {
			conn.WriteJSON(fiber.Map{"error": "User ID required"})
			return
		}

		mu.Lock()
		connectedUsers[userID] = conn
		mu.Unlock()
		log.Printf("User connected: %s", userID)
		broadcastOnlineUsers() // Broadcast after user connects

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		go pingRoutine(conn)

		for {
            _, msg, err := conn.ReadMessage()
            if err != nil {
                break
            }

            var baseMessage BaseMessage
            if err := json.Unmarshal(msg, &baseMessage); err != nil {
                log.Printf("Error parsing envelope: %v", err)
                continue
            }

            switch baseMessage.Type {
            case "MARK_MESSAGES_AS_SEEN":
                var payload MarkMessagesAsSeenPayload
                if err := json.Unmarshal(baseMessage.Payload, &payload); err != nil {
                    log.Printf("Error parsing MESSAGE_SEEN payload: %v", err)
                    continue
                }
                handleMessageSeen(conn, payload)

            case "TEXT":
                var payload TextMessagePayload
                if err := json.Unmarshal(baseMessage.Payload, &payload); err != nil {
                    log.Printf("Error parsing TEXT payload: %v", err)
                    continue
                }
                handleTextMessage(conn, payload)

            // Add more cases as needed

            default:
                log.Printf("Unknown message type: %s", baseMessage.Type)
            }
        }
    }))
}


func broadcastOnlineUsers() {
	mu.Lock()
	defer mu.Unlock()

	users := make([]string, 0, len(connectedUsers))
	for userID := range connectedUsers {
		users = append(users, userID)
	}

	connections := make([]*websocket.Conn, 0, len(connectedUsers))
	for _, conn := range connectedUsers {
		connections = append(connections, conn)
	}

	// Send to each connection in a goroutine to avoid blocking
	for _, conn := range connections {
		go func(c *websocket.Conn) {
			err := c.WriteJSON(fiber.Map{
				"type":  "ONLINE_USERS",
				"users": users,
			})
			if err != nil {
				log.Printf("Error sending online users: %v", err)
			}
		}(conn)
	}
}

func handleMessageSeen(conn *websocket.Conn, payload MarkMessagesAsSeenPayload) {
    session := database.Session.Db

    // Validate conversation exists
    var conversation database.Conversation
    if err := session.Where("id = ?", payload.ConversationID).First(&conversation).Error; err != nil {
        log.Printf("Error fetching conversation: %v", err)
        return
    }

    // Verify user is a conversation participant
    if conversation.Participant1ID != payload.UserID && conversation.Participant2ID != payload.UserID {
        log.Printf("User %s unauthorized for conversation %s", payload.UserID, payload.ConversationID)
        return
    }

    // Determine message sender (the other participant)
    otherParticipant := conversation.Participant1ID
    if otherParticipant == payload.UserID {
        otherParticipant = conversation.Participant2ID
    }

    var rowsAffected int64
    txErr := session.Transaction(func(tx *gorm.DB) error {
        result := tx.Model(&database.Message{}).
            Where("conversation_id = ? AND sender_id = ? AND seen = ?", 
                payload.ConversationID,
                otherParticipant,
                false).
            Update("seen", true)

        if result.Error != nil {
            return result.Error
        }
        rowsAffected = result.RowsAffected

        // Update conversation's last message seen status if applicable
        if conversation.LastMessageSenderID == payload.UserID {
            if err := tx.Model(&conversation).
                Update("last_message_seen", true).Error; err != nil {
                return err
            }
            log.Printf("Updated conversation %s last message seen status", payload.ConversationID)
        }

        log.Printf("Marked %d messages as seen in conversation %s by %s",
            rowsAffected,
            payload.ConversationID,
            payload.UserID)

        return nil
    })

    if txErr != nil {
        log.Printf("Message seen transaction failed: %v", txErr)
        return
    }

    // Notify the message sender only if messages were updated
    if rowsAffected > 0 {
        if recipientCon := GetSocket(otherParticipant); recipientCon != nil {
            err := recipientCon.WriteJSON(fiber.Map{
                "type": "MESSAGES_SEEN",
                "conversationId": payload.ConversationID,
            })
            if err != nil {
                log.Printf("WebSocket notify failed: %v", err)
            }
        }
    }
}


func handleTextMessage(conn *websocket.Conn, payload TextMessagePayload) {
    // Save message to database
    // ...

    // Forward to recipient
    if recipientCon := GetSocket(payload.RecipientID); recipientCon != nil {
        recipientCon.WriteJSON(fiber.Map{
            "type": "TEXT",
            "payload": fiber.Map{
                "text":           payload.Text,
                "senderId":       payload.SenderID,
                "conversationId": payload.ConversationID,
                "createdAt":      time.Now().Format(time.RFC3339),
            },
        })
    }
}


func pingRoutine(conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
			return
		}
	}
}
