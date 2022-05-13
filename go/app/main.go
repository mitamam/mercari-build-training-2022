package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const (
	ImgDir = "image"
)

type Response struct {
	Message string `json:"message"`
}

type Items struct {
	Items []Item `json:"items"`
}

type Item struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func addItem(c echo.Context) error {
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	c.Logger().Infof("Receive item: %s, %s", name, category)

	// Open database
	db, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
	if err != nil {
		return err
	}
	stmt, err := db.Prepare("INSERT INTO items(name, category) VALUES( ?, ? )")
	if err != nil {
		return err
	}
	defer stmt.Close()
	// Insert data to database
	_, err = stmt.Exec(name, category);
	if err != nil {
		return err
	}

	message := fmt.Sprintf("item received: %s, %s", name, category)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

func getItem(c echo.Context) error {
	// Open database
	db, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
	if err != nil {
		return err
	}
	defer db.Close()

	// Get name and category data from database
	rows, err := db.Query("SELECT name, category FROM items")
	if err != nil {
		return err
	}
	defer rows.Close()

	// Store data to Items struct
	var items Items
	var item Item
	for rows.Next() {
		err := rows.Scan(&item.Name, &item.Category);
		if err != nil {
			return err
		}
		items.Items = append(items.Items, item)
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	res := items
	return c.JSON(http.StatusOK, res)
}

func searchItem(c echo.Context) error {
	keyword := c.QueryParam("keyword")
	c.Logger().Infof("Search by: %s", keyword)

	// Open database
	db, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
	if err != nil {
		return err
	}
	defer db.Close()

	// Prepare query to search items by keyword
	stmt, err := db.Prepare("SELECT name, category FROM items WHERE name LIKE ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Search data from database
	keyword = "%" + keyword + "%"
	rows, err := stmt.Query(keyword)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Store search results to Items struct
	var items Items
	var item Item
	for rows.Next() {
		err := rows.Scan(&item.Name, &item.Category);
		if err != nil {
			return err
		}
		items.Items = append(items.Items, item)
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	res := items
	return c.JSON(http.StatusOK, res)
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("itemImg"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	front_url := os.Getenv("FRONT_URL")
	if front_url == "" {
		front_url = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{front_url},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.GET("/items", getItem)
	e.POST("/items", addItem)
	e.GET("/search", searchItem)
	e.GET("/image/:itemImg", getImg)

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
