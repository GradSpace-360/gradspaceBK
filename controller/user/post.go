package user

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"gradspaceBK/database"
	"gradspaceBK/middlewares"
)

func PostRoutes(base *fiber.Group) error {
	post := base.Group("/posts")
	post.Use(middlewares.AuthMiddleware)
	
	post.Post("/", CreatePost)
	post.Get("/", GetPosts)
	post.Post("/:id/like", ToggleLike)
	post.Post("/:id/comment", CreateComment)
	post.Delete("/:id", DeletePost)
	post.Get("/user/:username", GetUserPosts)
	
	return nil
}


func CreatePost(c *fiber.Ctx) error {
    userData := c.Locals("user_data").(jwt.MapClaims)
    userID := userData["user_id"].(string)

    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid form data",
        })
    }

    var imagePath string
    if imageFile, err := c.FormFile("image"); err == nil {
        fileExt := filepath.Ext(imageFile.Filename)
        newFileName := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), uuid.New().String(), fileExt)
        uploadDir := "./uploads/post"
        
        if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to create upload directory",
            })
        }
        
        savePath := filepath.Join(uploadDir, newFileName)
        if err := c.SaveFile(imageFile, savePath); err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to save image",
            })
        }
        imagePath = savePath
    }

    content := form.Value["content"][0]
    
    newPost := database.Post{
        AuthorID: userID,
        Content:  &content,
        Image:    &imagePath,
    }

    session := database.Session.Db
    if err := session.Create(&newPost).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to create post",
        })
    }

    postData := map[string]interface{}{
        "id":        newPost.ID,
        "content":   *newPost.Content,
        "image":     *newPost.Image,
        "createdAt": newPost.CreatedAt,
        "author": map[string]interface{}{
            "id":       newPost.AuthorID,
            "username": getUsername(newPost.AuthorID), 
            "image":    getProfileImage(newPost.AuthorID),
        },
        "comments":  0, 
        "likes":     0, 
        "isLiked":   false, 
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "success": true,
        "data":    postData,
    })
}

func getUsername(userID string) string {
    var user database.User
    database.Session.Db.
        Model(&database.User{}).
        Where("id = ?", userID).
        First(&user)
    
    return *user.UserName
}


func GetPosts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	session := database.Session.Db
	
	var totalItems int64
	session.Model(&database.Post{}).Count(&totalItems)
	totalPages := (int(totalItems) + limit - 1) / limit

	var posts []database.Post
	result := session.Preload("Author").
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC").Preload("Author")
		}).
		Preload("Likes").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch posts",
		})
	}

	var response []map[string]interface{}
	for _, post := range posts {
		
		var commentList []map[string]interface{}
		for _, comment := range post.Comments {
			commentData := map[string]interface{}{
				"id":        comment.ID,
				"content":   comment.Content,
				"createdAt": comment.CreatedAt,
				"author": map[string]interface{}{
					"id":       comment.Author.ID,
					"username": comment.Author.UserName,
					"image":    getProfileImage(comment.Author.ID),
				},
			}
			commentList = append(commentList, commentData)
		}

		postData := map[string]interface{}{
			"id":        post.ID,
			"content":   post.Content,
			"image":     post.Image,
			"createdAt": post.CreatedAt,
			"author": map[string]interface{}{
				"id":       post.Author.ID,
				"username": post.Author.UserName,
				"image":    getProfileImage(post.Author.ID),
			},
			"comments":     len(post.Comments),
			"likes":        len(post.Likes),
			"isLiked":      isPostLikedByUser(c, post.ID),
			"commentList": commentList, 
		}
		response = append(response, postData)
	}

	return c.JSON(fiber.Map{
		"data": response,
		"meta": map[string]interface{}{
			"current_page": page,
			"per_page":     limit,
			"total_pages":  totalPages,
			"total_items":  totalItems,
		},
	})
}

func getProfileImage(userID string) string {
	var profile database.UserProfile
	database.Session.Db.
		Model(&database.UserProfile{}).
		Where("user_id = ?", userID).
		First(&profile)
	
	return profile.ProfileImage
}

func ToggleLike(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)
	postID := c.Params("id")

	session := database.Session.Db

	var existingLike database.Like
	if err := session.Where("user_id = ? AND post_id = ?", userID, postID).First(&existingLike).Error; err == nil {
		if err := session.Delete(&existingLike).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to remove like",
			})
		}
	} else {
		newLike := database.Like{
			PostID: postID,
			UserID: userID,
		}
		if err := session.Create(&newLike).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to add like",
			})
		}

		// Create notification (if not own post)
		var post database.Post
		session.First(&post, "id = ?", postID)
		if post.AuthorID != userID {
			notification := database.Notification{
				UserID:    post.AuthorID,
				CreatorID: userID,
				Type:      database.NotificationTypeLike,
				PostID:    &postID,
			}
			if err := session.Create(&notification).Error; err != nil {
						fmt.Println("Failed to create notification:", err)
			}
		}
	}
	return c.JSON(fiber.Map{"success": true})
}

func CreateComment(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)
	postID := c.Params("id")

	type CommentRequest struct {
		Content string `json:"content"`
	}
	
	var req CommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	session := database.Session.Db

	comment := database.Comment{
		Content:  req.Content,
		AuthorID: userID,
		PostID:   postID,
	}
	if err := session.Create(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create comment",
		})
	}

	 var createdComment database.Comment
	 if err := session.Preload("Author").
		 First(&createdComment, "id = ?", comment.ID).
		 Error; err != nil {
		 return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			 "error": "Failed to fetch created comment",
		 })
	 }
 
    commentResponse := map[string]interface{}{
        "id":        createdComment.ID,
        "content":   createdComment.Content,
        "createdAt": createdComment.CreatedAt,
        "author": map[string]interface{}{
            "id":       createdComment.Author.ID,
            "username": createdComment.Author.UserName,
            "image":    getProfileImage(createdComment.Author.ID), // Use existing helper
        },
    }

	var post database.Post
	session.First(&post, "id = ?", postID)
	if post.AuthorID != userID {
		notification := database.Notification{
			UserID:    post.AuthorID,
			CreatorID: userID,
			Type:      database.NotificationTypeComment,
			PostID:    &postID,
			CommentID: &comment.ID,
		}
		if err := session.Create(&notification).Error; err != nil {
			fmt.Println("Failed to create notification:", err)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    commentResponse,
	})
}

func DeletePost(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)
	postID := c.Params("id")

	session := database.Session.Db

	var post database.Post
	if err := session.First(&post, "id = ?", postID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Post not found",
		})
	}

	if post.AuthorID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := session.Delete(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete post",
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

func isPostLikedByUser(c *fiber.Ctx, postID string) bool {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)

	var like database.Like
	result := database.Session.Db.
		Where("user_id = ? AND post_id = ?", userID, postID).
		First(&like)

	return result.Error == nil
}


func GetUserPosts(c *fiber.Ctx) error {
    username := c.Params("username")
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit, _ := strconv.Atoi(c.Query("limit", "10"))

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 10
    }
    offset := (page - 1) * limit

    session := database.Session.Db

    var user database.User
    if err := session.Where("user_name = ?", username).First(&user).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found",
        })
    }

    var totalItems int64
    session.Model(&database.Post{}).Where("author_id = ?", user.ID).Count(&totalItems)
    totalPages := (int(totalItems) + limit - 1) / limit

    var posts []database.Post
    result := session.Where("author_id = ?", user.ID).
        Preload("Comments", func(db *gorm.DB) *gorm.DB {
            return db.Order("created_at ASC").Preload("Author")
        }).
        Preload("Likes").
        Preload("Author").
        Order("created_at DESC").
        Offset(offset).
        Limit(limit).
        Find(&posts)

    if result.Error != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to fetch posts",
        })
    }


    var response []map[string]interface{}
    for _, post := range posts {
        var commentList []map[string]interface{}
        for _, comment := range post.Comments {
            commentData := map[string]interface{}{
                "id":        comment.ID,
                "content":   comment.Content,
                "createdAt": comment.CreatedAt,
                "author": map[string]interface{}{
                    "id":       comment.Author.ID,
                    "username": comment.Author.UserName,
                    "image":    getProfileImage(comment.Author.ID),
                },
            }
            commentList = append(commentList, commentData)
        }

        postData := map[string]interface{}{
            "id":        post.ID,
            "content":   post.Content,
            "image":     post.Image,
            "createdAt": post.CreatedAt,
            "author": map[string]interface{}{
                "id":       user.ID,
                "username": user.UserName,
                "image":    getProfileImage(user.ID),
            },
            "comments":    len(post.Comments),
            "likes":       len(post.Likes),
            "isLiked":      isPostLikedByUser(c, post.ID),
            "commentList": commentList,
        }
        response = append(response, postData)
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data":    response,
        "meta": map[string]interface{}{
            "current_page": page,
            "per_page":     limit,
            "total_pages":  totalPages,
            "total_items":  totalItems,
        },
    })
}