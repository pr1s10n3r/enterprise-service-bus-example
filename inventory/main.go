package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var database *sql.DB

type Item struct {
	ID        int64     `xml:"id"`
	Name      string    `xml:"name"`
	Stock     int64     `xml:"stock"`
	CreatedAt time.Time `xml:"created_at,omitempty"`
	UpdatedAt time.Time `xml:"updated_at,omitempty"`
}

func addItem(ctx *gin.Context) {
	item := new(Item)

	if err := ctx.BindXML(item); err != nil {
		return
	}

	stmt, _ := database.Prepare("INSERT INTO items (name, stock) VALUES (?, ?)")
	defer stmt.Close()

	result, err := stmt.Exec(item.Name, item.Stock)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	item.ID, _ = result.LastInsertId()

	ctx.XML(http.StatusCreated, item)
}

func getItems(ctx *gin.Context) {
	offset, err := strconv.ParseUint(getParamOr(ctx, "offset", "0"), 10, 64)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	limit, err := strconv.ParseUint(getParamOr(ctx, "limit", "50"), 10, 64)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	stmt, err := database.Prepare("SELECT id, name, stock, created_at, updated_at FROM items LIMIT ?, ?")
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(offset, limit)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	items := make([]Item, 0)
	for rows.Next() {
		item := Item{}

		err := rows.Scan(&item.ID, &item.Name, &item.Stock, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		items = append(items, item)
	}

	ctx.XML(http.StatusOK, items)
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	initDatabase()

	router := gin.Default()

	items := router.Group("/items")
	items.POST("/", addItem)
	items.GET("/", getItems)

	router.Run(":3060")
}

func initDatabase() {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?parseTime=true",
		os.Getenv("MYSQL_USERNAME"),
		os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("MYSQL_HOST"),
		os.Getenv("MYSQL_DATABASE"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		panic(err)
	}

	database = db
}

func getParamOr(ctx *gin.Context, key, alt string) string {
	value := ctx.Query(key)
	if value == "" {
		return alt
	}
	return value
}
