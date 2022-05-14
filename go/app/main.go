package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/mattn/go-sqlite3"
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
	Image    string `json:"image"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func addItem(c echo.Context) error {
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	image := c.FormValue("image")
	c.Logger().Infof("Receive item: %s, %s, %s", name, category, image)

	// Open an image
	f, err := os.Open(image)
	if err != nil {
		return err
	}
	defer f.Close()

	// hash an image with SHA-256
	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return err
	}
	sum := hex.EncodeToString(h.Sum(nil)) + ".jpg"

	// Open the image again
	f, err = os.Open(image)
	if err != nil {
		return err
	}
	defer f.Close()

	// Create an image
	i, err := os.Create("./images/" + sum)
	if err != nil {
		return err
	}
	defer i.Close()

	// Decode and encode an image
	img, err := jpeg.Decode(f)
	if err != nil {
		return err
	}
	err = jpeg.Encode(i, img, nil)
	if err != nil {
		return err
	}

	// Open database
	db, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
	if err != nil {
		return err
	}
	stmt, err := db.Prepare("INSERT INTO items(name, category, image) VALUES( ?, ?, ? )")
	if err != nil {
		return err
	}
	defer stmt.Close()
	// Insert data to database
	_, err = stmt.Exec(name, category, sum)
	if err != nil {
		return err
	}

	message := fmt.Sprintf("item received: %s, %s, %s", name, category, image)
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
	rows, err := db.Query("SELECT name, category, image FROM items")
	if err != nil {
		return err
	}
	defer rows.Close()

	// Store data to Items struct
	var items Items
	var item Item
	for rows.Next() {
		err := rows.Scan(&item.Name, &item.Category, &item.Image)
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

func getItemById(c echo.Context) error {
	itemId := c.Param("itemId")

	// Open database
	db, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
	if err != nil {
		return err
	}
	defer db.Close()

	// Search an item by id from database
	var item Item
	err = db.QueryRow("SELECT name, category, image FROM items WHERE id = ?", itemId).Scan(&item.Name, &item.Category, &item.Image)
	if err != nil {
		return err
	}

	res := item
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
		err := rows.Scan(&item.Name, &item.Category)
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
	e.GET("/items/:itemId", getItemById)
	e.POST("/items", addItem)
	e.GET("/search", searchItem)
	e.GET("/image/:itemImg", getImg)

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
