package admin

import (
	"fmt"
	"gradspaceBK/database"
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TODO: create middleware for adminAuthentication for improving security
// group.Use(middlewares.AdminMiddleware)

// Admin routes (separate admin router group)
func AdminPostRoutes(base *fiber.Group) {
        group := base.Group("/admin/post-reports")
        group.Get("/", GetReportedPosts)           // List reported posts
        group.Get("/:postId", GetPostReportDetails) // Detailed view
        group.Delete("/:postId/dismiss", DismissPostReports)
        group.Delete("/posts/:postId", AdminDeletePost)
}

func GetReportedPosts(c *fiber.Ctx) error {
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit, _ := strconv.Atoi(c.Query("limit", "10"))

    if page < 1 { page = 1 }
    if limit < 1 || limit > 100 { limit = 10 }
    offset := (page - 1) * limit

    // Raw SQL for aggregation
    query := `
        SELECT 
            post_id,
            COUNT(*) as flag_count,
            MAX(reason) as top_reason,
            MAX(created_at) as last_flagged
        FROM post_reports
        GROUP BY post_id
        ORDER BY last_flagged DESC
        LIMIT ? OFFSET ?
    `

    var reports []struct {
        PostID      string    `gorm:"column:post_id"`
        FlagCount   int       `gorm:"column:flag_count"`
        TopReason   string    `gorm:"column:top_reason"`
        LastFlagged time.Time `gorm:"column:last_flagged"`
    }

    database.Session.Db.Raw(query, limit, offset).Scan(&reports)

    // Get post details for each report
    var response []map[string]interface{}
    for _, r := range reports {
        var post database.Post
        database.Session.Db.
            Preload("Author").
            First(&post, "id = ?", r.PostID)

        response = append(response, map[string]interface{}{
            "post_id": r.PostID,
            "post_preview": post.Image,
            "author": map[string]interface{}{
                "id": post.AuthorID,
                "username": post.Author.UserName,
            },
            "flag_count": r.FlagCount,
            "top_reason": r.TopReason,
            "last_flagged": r.LastFlagged,
            "post_date": post.CreatedAt,
        })
    }

    // Get total count
    var total int64
    database.Session.Db.Model(&database.PostReport{}).
        Select("COUNT(DISTINCT post_id)").
        Scan(&total)

    return c.JSON(fiber.Map{
        "data": response,
        "meta": map[string]interface{}{
            "current_page": page,
            "per_page": limit,
            "total_pages": int(math.Ceil(float64(total) / float64(limit))),
            "total_items": total,
        },
    })
}

func GetPostReportDetails(c *fiber.Ctx) error {
    postID := c.Params("postId")

    // Get post data
    var post database.Post
    if err := database.Session.Db.
        Preload("Author").
        Preload("Comments").
        Preload("Likes").
        First(&post, "id = ?", postID).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Post not found",
        })
    }

    // Get all reports for this post
    var reports []database.PostReport
    database.Session.Db.
        Preload("Reporter").
        Where("post_id = ?", postID).
        Find(&reports)

    // Format flag data
    var flagData []map[string]interface{}
    for _, r := range reports {
        flagData = append(flagData, map[string]interface{}{
            "reporter": r.Reporter.UserName,
            "reason": r.Reason,
            "flagged_at": r.CreatedAt,
        })
    }

    return c.JSON(fiber.Map{
        "post_data": map[string]interface{}{
            "id": post.ID,
            "content": post.Content,
            "image": post.Image,
            "createdAt": post.CreatedAt,
            "author": map[string]interface{}{
                "id": post.Author.ID,
                "username": post.Author.UserName,
            },
            "likes": len(post.Likes),
            "comments": len(post.Comments),
        },
        "flag_data": flagData,
    })
}

func DismissPostReports(c *fiber.Ctx) error {
    postID := c.Params("postId")

    result := database.Session.Db.
        Where("post_id = ?", postID).
        Delete(&database.PostReport{})

    if result.Error != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to dismiss reports",
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "message": fmt.Sprintf("Dismissed %d reports", result.RowsAffected),
    })
}

func AdminDeletePost(c *fiber.Ctx) error {
    postID := c.Params("postId")

    // Delete post and cascade delete related data
    result := database.Session.Db.
        Select("Comments", "Likes", "PostReports").
        Delete(&database.Post{BaseModel: database.BaseModel{ID: postID}})

    if result.Error != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to delete post",
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "message": "Post and associated data deleted",
    })
}