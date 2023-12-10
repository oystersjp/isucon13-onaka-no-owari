package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
)

type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type TagCache struct {
	mu       sync.RWMutex
	tags     map[int64]TagModel
	nameToID map[string]int64
}

type TagModel struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

type TagsResponse struct {
	Tags []*Tag `json:"tags"`
}

var tagCache = NewTagCache()

func NewTagCache() *TagCache {
	return &TagCache{
		tags:     make(map[int64]TagModel),
		nameToID: make(map[string]int64),
	}
}

// GetTagByID は指定されたIDのタグをキャッシュから取得します。
func (c *TagCache) GetTagByID(id int64) (TagModel, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tag, found := c.tags[id]
	return tag, found
}

// GetTagIDByName は指定された名前のタグIDをキャッシュから取得します。
func (c *TagCache) GetTagIDByName(name string) (int64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	id, found := c.nameToID[name]
	return id, found
}

func initTagCache(c echo.Context) error {
	ctx := c.Request().Context()

	tx, err := dbConn.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var tagModels []*TagModel
	if err := tx.SelectContext(ctx, &tagModels, "SELECT * FROM tags"); err != nil {
		return fmt.Errorf("failed to get tags: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	tagCache.mu.Lock()
	defer tagCache.mu.Unlock()

	for _, tagModel := range tagModels {
		tagCache.tags[tagModel.ID] = *tagModel
		tagCache.nameToID[tagModel.Name] = tagModel.ID
	}

	return c.JSON(http.StatusOK, &TagsResponse{})
}

func getTagHandler(c echo.Context) error {
	tagCache.mu.RLock()
	defer tagCache.mu.RUnlock()
	tags := make([]*Tag, 0, len(tagCache.tags))
	for _, tagModel := range tagCache.tags {
		tags = append(tags, &Tag{
			ID:   tagModel.ID,
			Name: tagModel.Name,
		})
	}

	return c.JSON(http.StatusOK, &TagsResponse{
		Tags: tags,
	})
}

// 配信者のテーマ取得API
// GET /api/user/:username/theme
func getStreamerThemeHandler(c echo.Context) error {
	ctx := c.Request().Context()

	if err := verifyUserSession(c); err != nil {
		// echo.NewHTTPErrorが返っているのでそのまま出力
		c.Logger().Printf("verifyUserSession: %+v\n", err)
		return err
	}

	username := c.Param("username")

	tx, err := dbConn.BeginTxx(ctx, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to begin transaction: "+err.Error())
	}
	defer tx.Rollback()

	userModel := UserModel{}
	err = tx.GetContext(ctx, &userModel, "SELECT id FROM users WHERE name = ?", username)
	if errors.Is(err, sql.ErrNoRows) {
		return echo.NewHTTPError(http.StatusNotFound, "not found user that has the given username")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user: "+err.Error())
	}

	themeModel := ThemeModel{}
	if err := tx.GetContext(ctx, &themeModel, "SELECT * FROM themes WHERE user_id = ?", userModel.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user theme: "+err.Error())
	}

	if err := tx.Commit(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to commit: "+err.Error())
	}

	theme := Theme{
		ID:       themeModel.ID,
		DarkMode: themeModel.DarkMode,
	}

	return c.JSON(http.StatusOK, theme)
}
